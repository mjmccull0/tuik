package main

import (
  "strings"
	"os/exec"
  "tuik/components"
)

type Navigator struct {
	Views          map[string]*components.View
  ActiveViewID     string
	Context components.Context
	
}

type NavResult struct {
  Command  string
  NextViewID string
  IsUpdate bool
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
    
    // HYDRATE: If any list has a command string, run it now
    for i, child := range view.Children {
        if l, ok := child.(*components.List); ok {
            if cmd, isCmd := l.Input.(string); isCmd {
                // Resolve the command (in case it uses {{.variables}})
                resolvedCmd := n.Context.Resolve(cmd)
                
                // Execute and split into items
                out, _ := exec.Command("sh", "-c", resolvedCmd).Output()
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
                l.Input = items // Swap string for the actual list
                view.Children[i] = l
            }
        }
    }

    return view, n.Context
}
