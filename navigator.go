package main

import (
	"fmt"
	"strings"
	"encoding/json"
)

type Navigator struct {
	Config TuikConfig
	Data   map[string]string
	Styles map[string]string
}


type Handler struct {
	Stdout string          `json:"stdout,omitempty"`
	Success any            `json:"success,omitempty"`
	Mapping map[string]any `json:"-"`
}


// Resolve replaces {{.key}} with values from state or {{style.key}} with ANSI codes
func (n *Navigator) Resolve(input string) string {
	output := input

	// 1. Resolve Styles: {{.styles.key}}
	// We do this FIRST so style codes can be embedded in data strings
	for k, v := range n.Config.Styles {
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
	// Convert any type (int, bool, string) to string safely
	strVal := fmt.Sprintf("%v", value)
	// We resolve it so that if we set A = "{{.B}}", A gets the current value of B
	n.Data[key] = n.Resolve(strVal)
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
				// We resolve the value so you can pass templates 
				// like "confirm_msg": "Edit {{.selected_path}}?"
				resolvedVal := n.Resolve(fmt.Sprintf("%v", val))
				n.Set(key, resolvedVal)
			}
		}

		return n.Resolve(target)

	default:
		return ""
	}
}

func (n *Navigator) HydrateView(view View) {
	// Process local view variables first
	for key, val := range view.StateSet {
		// Resolve ensures we can use styles or other state in our local variables
		// e.g., "header": "Files for {{.user}}"
		resolved := n.Resolve(fmt.Sprintf("%v", val))
		n.Set(key, resolved)
	}
}

// GetView retrieves a view definition and ensures the Navigator's 
// own config is the source of truth.
func (n *Navigator) GetView(id string) (View, bool) {
	view, ok := n.Config.Views[id]
	return view, ok
}

func (n *Navigator) GetMainId() string {
	return n.Config.Main
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
