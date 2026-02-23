package main

import (
  // "strings"
  "tuik/components"
)

type Navigator struct {
	Views          map[string]components.View
  ActiveViewID     string
	Context components.Context
	
}

type NavResult struct {
  NextView string
  Command  string
  IsUpdate bool
}

func (n *Navigator) ProcessAction(action string, data map[string]string) NavResult {
	// 1. Interpolate first (replaces {{.vars}} in the action string)
	interpolated := interpolate(action, data)

	// 2. Check if it's a view swap
	// We check n.Views (our Registry) and update n.ActiveViewID
	if _, exists := n.Views[interpolated]; exists {
		n.ActiveViewID = interpolated
		return NavResult{NextView: interpolated, IsUpdate: true}
	}

	// 3. Otherwise, it's a shell command (git commit, etc.)
	return NavResult{Command: interpolated, IsUpdate: false}
}

func (n *Navigator) GetActiveView() (components.View, components.Context) {
    return n.Views[n.ActiveViewID], n.Context
}
