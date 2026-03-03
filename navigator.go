package main

import (
	"fmt"
  "strings"
	"os/exec"
  "tuik/components"
)

type Config struct {
  Main    string
  ShellFunctions string 
}

type Navigator struct {
	Views      map[string]*components.View
  ActiveViewID   string
	Context    components.Context
	Config     Config
}

type NavResult struct {
  Command  string
  NextViewID string
  IsUpdate bool
}

func (n *Navigator) execWrapped(cmd string) ([]byte, error) {
	// Resolve any {{.variables}} first
	resolved := n.Context.Resolve(cmd)

	// If a setup script is defined, prepend the source command
	finalCmd := resolved
	if n.Config.ShellFunctions != "" {
		// We use ';' to ensure the command runs even if the source has a minor issue
		finalCmd = fmt.Sprintf("source %s; %s", n.Config.ShellFunctions, resolved)
	}

	// Use zsh explicitly if your functions are zsh-specific
	return exec.Command("zsh", "-c", finalCmd).Output()
}

func (n *Navigator) ProcessAction(action string, data map[string]string) NavResult {
	// We check n.Views (our Registry) and update n.ActiveViewID
  if strings.HasPrefix(action, "view:") {
		target := strings.TrimPrefix(action, "view:")
		n.ActiveViewID = target
		return NavResult{NextViewID: target, IsUpdate: true}
	}

	// 3. Otherwise, it's a shell command (git commit, etc.)
	return NavResult{Command: action}
}

func (n *Navigator) GetActiveView() (*components.View, components.Context) {
	view := n.Views[n.ActiveViewID]
	
	for i, child := range view.Children {
		if l, ok := child.(*components.List); ok {
			if cmd, isCmd := l.Input.(string); isCmd {
				// Use your wrapper! It handles Resolution and Sourcing automatically.
				out, _ := n.execWrapped(cmd) 
				
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				
				var items []components.ListItem
				for _, line := range lines {
					if line != "" {
						items = append(items, components.ListItem{
							Text:    line, 
							OnPress: line,
						})
					}
				}
				l.Input = items 
				view.Children[i] = l
			}
		}
	}

	return view, n.Context
}
