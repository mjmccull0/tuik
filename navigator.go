package main

import (
	"fmt"
	"strings"
)

type Navigator struct {
	Config TuikConfig
	Data   map[string]string
	Styles map[string]string
}

// Resolve replaces {{.key}} with values from state or {{style.key}} with ANSI codes
func (n *Navigator) Resolve(input string) string {
	output := input

	// 1. Resolve Styles: {{.styles.key}}
	// We do this FIRST so style codes can be embedded in data strings
	for k, v := range n.Styles {
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
func (n *Navigator) DetermineNext(rule any) string {
	// Case 1: Simple string jump ("on_success": "view:hub")
	if target, ok := rule.(string); ok {
		return n.Resolve(target)
	}

	// Case 2: Map-based logic
	outcome, ok := rule.(map[string]any)
	if !ok {
		return "exit"
	}

	var rawTarget any

	// Is it a Handoff? (Contains a direct "target" key)
	if target, ok := outcome["target"]; ok {
		rawTarget = target
		// Apply state changes defined at the handoff level
		n.ApplyState(outcome)
	} else {
		// It's a Branching Menu (keys match the last command's output)
		choice := n.Data["last_output"]
		if branch, ok := outcome[choice]; ok {
			rawTarget = branch
		}
	}

	// Now we see what the rawTarget actually is
	if rawTarget == nil {
		return "exit"
	}

	// If the result of the branch was a string, return it
	if t, ok := rawTarget.(string); ok {
		return n.Resolve(t)
	}

	// If the result was another map, apply its state and return its target
	if tMap, ok := rawTarget.(map[string]any); ok {
		n.ApplyState(tMap)
		if target, ok := tMap["target"].(string); ok {
			return n.Resolve(target)
		}
	}

	return "exit"
}
