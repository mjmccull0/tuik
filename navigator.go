package main

import (
	"fmt"
	"strings"
	"encoding/json"
	"os/exec"
)

type Navigator struct {
	Tuik   Tuik 
	Data   map[string]string
}


type Handler struct {
	Stdout string          `json:"stdout,omitempty"`
	Success any            `json:"success,omitempty"`
	Mapping map[string]any `json:"-"`
}


func (n *Navigator) InitialLoad() {
    // If the user hasn't defined a 'get' hook, there's nothing to load
    getCmd := n.Tuik.Config.State.Get
    if getCmd == "" {
        return
    }

    // Logic: How do we know WHICH keys to get?
    // Option A: We run a 'list' command if provided.
    // Option B: We wait for a view to request a key (Lazy Load).
    
    // For now, let's look at a "Pre-emptive Load" of common keys 
    // or a specific 'list' command if we add it to StateHooks.
}

// Resolve replaces {{.key}} with values from state or {{style.key}} with ANSI codes
func (n *Navigator) Resolve(input string) string {
	output := input

	// 1. Resolve Styles: {{.styles.key}}
	// We do this FIRST so style codes can be embedded in data strings
	for k, v := range n.Tuik.Styles {
		placeholder := fmt.Sprintf("{{.styles.%s}}", k) // Matches your JSON
		output = strings.ReplaceAll(output, placeholder, v)
	}

	// 2. Resolve State Variables: {{.variable_name}}
	for k, v := range n.Data {
		placeholder := fmt.Sprintf("{{.%s}}", k)
		output = strings.ReplaceAll(output, placeholder, v)
	}

	return output
}

// Set updates a single key in the state, ensuring the value is resolved
func (n *Navigator) Set(key string, value any) {
  strVal := fmt.Sprintf("%v", value)
	resolvedVal := n.Resolve(strVal)
	
	n.Data[key] = resolvedVal

	// PERSISTENCE
	// Use the renamed Tuik.Settings path
	persistCmd := n.Tuik.Config.State.Set
	if persistCmd != "" {
		cmd := strings.ReplaceAll(persistCmd, "{{.key}}", key)
		cmd = strings.ReplaceAll(cmd, "{{.value}}", resolvedVal)

		go func(c string) {
			_ = exec.Command("sh", "-c", c).Run()
		}(cmd)
	}
}

// ApplyState takes a map (like a transition object) and processes its "set_state" block
func (n *Navigator) ApplyState(input any) {
	m, ok := input.(map[string]any)
	if !ok {
		return
	}

	if newState, ok := m["set_state"].(map[string]any); ok {
		for k, v := range newState {
			n.Set(k, v)
		}
	}
}

// DetermineNext parses the OnSuccess interface to find the next view string
func (n *Navigator) DetermineNext(input any) string {
	if input == nil {
		return ""
	}

	switch v := input.(type) {
	case string:
		// Simple case: "view:confirm"
		return n.Resolve(v)

	case map[string]any:
		// Smart Target case: {"target": "view:confirm", "state.set": {...}}
		target, _ := v["target"].(string)

		// Apply the "Transition Primes"
		if primes, ok := v["state.set"].(map[string]any); ok {
			for key, val := range primes {
				n.Set(key, val)
			}
		}

		return n.Resolve(target)

	default:
		return ""
	}
}

func (n *Navigator) HydrateView(view View) {
	// 1. THE MISSING LINK: Handle state.get
	// This maps: "hello": "test_var" -> n.Data["hello"] = $(skate get test_var)
	getTemplate := n.Tuik.Config.State.Get 
	if view.Config.State.Get != "" {
		getTemplate = view.Config.State.Get
	}

	for localKey, externalKey := range view.StateGet {
		if getTemplate != "" {
			cmdStr := strings.ReplaceAll(getTemplate, "{{.key}}", externalKey)
			
			// Execute synchronously so Resolve() has the data immediately
			out, err := exec.Command("sh", "-c", cmdStr).Output()
			if err == nil {
				n.Data[localKey] = strings.TrimSpace(string(out))
			}

			if err != nil {
				// If skate isn't in the PATH or the command fails
				fmt.Printf("DEBUG: Skate Error: %v\n", err) 
			}
			fmt.Printf("DEBUG: Fetched %s -> %s\n", cmdStr, string(out))
		}
	}

	// 2. Handle local state.set
	for key, val := range view.StateSet {
		resolved := n.Resolve(fmt.Sprintf("%v", val))
		n.Set(key, resolved)
	}
}

// GetView retrieves a view definition and ensures the Navigator's 
// own config is the source of truth.
func (n *Navigator) GetView(id string) (View, bool) {
	view, ok := n.Tuik.Views[id]
	return view, ok
}

func (n *Navigator) GetMainId() string {
	return n.Tuik.Main
}

// Custom Unmarshaler to catch the dynamic keys
func (h *Handler) UnmarshalJSON(data []byte) error {
	type alias Handler
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*h = Handler(aux)

	// Now catch the "wildcard" keys
	var raw map[string]any
	json.Unmarshal(data, &raw)
	delete(raw, "stdout")
	delete(raw, "success")
	h.Mapping = raw
	return nil
}
