package main

import (
	"fmt"
  "strings"
	"os/exec"
  "tuik/components"
  tea "github.com/charmbracelet/bubbletea"
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

func (n *Navigator) Refresh() tea.Cmd {
	return func() tea.Msg {
		return RefreshMsg{}
	}
}

func (n *Navigator) InitView(viewID string) (*components.View, tea.Cmd) {
	view, ok := n.Views[viewID]
	if !ok {
		return nil, nil
	}

	n.ActiveViewID = viewID
	var cmds []tea.Cmd

	for i := range view.Children {
		// child is a components.Component (Interface)
		child := view.Children[i]

		// Use a type switch to access specific fields
		switch c := child.(type) {
		case *components.List: // Assuming your struct name
			// Assert that Input is a string before checking if it's a shell command
			if inputStr, ok := c.Input.(string); ok && inputStr != "" {
					if n.isShellCommand(inputStr) {
							cmds = append(cmds, n.runListHydration(c))
					}
			}
		}
	}

	return view, tea.Batch(cmds...)
}

func (n *Navigator) runListHydration(l *components.List) tea.Cmd {
    return func() tea.Msg {
        // Assert Input is a string here as well
        if inputStr, ok := l.Input.(string); ok {
            return components.ListHydrationMsg{
                ID:    l.ID,
                Items: n.executeShellAndParse(inputStr),
            }
        }
        return nil
    }
}

// This might warrant a json config option
func (n *Navigator) isShellCommand(input string) bool {
	// Simple heuristic: if it contains a space or starts with a known tool
	keywords := []string{"git", "project", "ls", "./", "get_"}
	for _, k := range keywords {
		if strings.HasPrefix(input, k) {
			return true
		}
	}
	return strings.Contains(input, " ")
}

func (n *Navigator) executeShellAndParse(cmdStr string) []components.ListItem {
	// Use your existing prepareCmd which handles the zsh + source setup
	c := n.prepareCmd(cmdStr)
	out, err := c.CombinedOutput()
	if err != nil {
		return []components.ListItem{{Text: "Error running command"}}
	}

	lines := strings.Split(string(out), "\n")
	var items []components.ListItem
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			items = append(items, components.ListItem{Text: trimmed})
		}
	}
	return items
}
