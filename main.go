package main

import (
  "fmt"
	"log"
  "os"
  "os/exec"
  "strings"

  "github.com/charmbracelet/lipgloss"
  "github.com/charmbracelet/bubbles/textinput"
  tea "github.com/charmbracelet/bubbletea"

  "tuik/components"
  "tuik/parser"
	"tuik/utils"
)

type model struct {
  navigator *Navigator
	width     int
	height    int
	logger    *log.Logger
}

// getContextData gathers all current component values into a map for the navigator
func (m model) getContextData() map[string]string {
    data := make(map[string]string)
    // You likely have a loop here that visits every component 
    // and calls GetValue(), similar to your existing sync logic.
    return data
}

// executeShellCommand runs the resolved string (e.g., "git commit...") in the terminal
func (m model) executeShellCommand(command string) tea.Cmd {
    return func() tea.Msg {
        // Using exec.Command to actually run the git logic
        cmd := exec.Command("sh", "-c", command)
        output, err := cmd.CombinedOutput()
        
        if err != nil {
            m.logger.Printf("Shell Error: %v, Output: %s", err, string(output))
        } else {
            m.logger.Printf("Shell Success: %s", string(output))
        }
        return nil // Or a 'SuccessMsg' if you want to show a toast
    }
}

func (m *model) syncContext() {
	view, _ := m.navigator.GetActiveView()
	// Ensure the data map is initialized
	if m.navigator.Context.Data == nil {
		m.navigator.Context.Data = make(map[string]string)
	}
	
	// Use recursion to find data in nested components (like Lists inside Boxes)
	for _, child := range view.Children {
		m.extractData(child)
	}
}

func (m *model) extractData(c components.Component) {
    if c == nil { return }

    // If it's a Box, drill down into its children
    if c.GetType() == "box" {
        if box, ok := c.(*components.Box); ok {
            for _, sub := range box.Children {
                m.extractData(sub)
            }
            return
        }
    }

    // If the component has an ID (like "selected_file"), save its value
    if id := c.GetID(); id != "" {
        val := c.GetValue()
        m.navigator.Context.Data[id] = val
        // This will now show up in tuik.log
        utils.Log("Synced: %s = %s", id, val)
    }
}

// New helper to handle nested components (Boxes inside Boxes)
func (m *model) pullComponentData(c components.Component) {
    // 1. If it's a Box, look at its children
    if c.GetType() == "box" {
        // We need to type-assert to get to the Children slice
        if box, ok := c.(*components.Box); ok {
            for _, subChild := range box.Children {
                m.pullComponentData(subChild)
            }
        }
        return
    }

    // 2. If it's a data component with an ID, grab the value
    if id := c.GetID(); id != "" {
        m.navigator.Context.Data[id] = c.GetValue()
    }
}

func initialModel(cfg components.Config) model {
	// 1. Create a clean map for the Views
	viewMap := make(map[string]components.View)
	for k, v := range cfg.Views {
		viewMap[k] = *v // Convert pointer to value if needed
	}

	// Initialize the Navigator with an empty data bucket
	nav := &Navigator{
		Views:        viewMap,
		ActiveViewID: cfg.Main,
		Context: components.Context{
			Data:   make(map[string]string),
			Styles: make(map[string]lipgloss.Style),
		},
	}

	// Initial focus logic: We tell the current view to focus its first focusable child
	if view, ok := nav.Views[nav.ActiveViewID]; ok {
		for i := range view.Children {
			if view.Children[i].IsFocusable() {
				view.Children[i].Focus()
				// Update the view back in the registry since Focus() might change internal state
				nav.Views[nav.ActiveViewID] = view
				break
			}
		}
	}

	m := model{navigator: nav}
	m.syncContext()
	return m
}

func (m model) Init() tea.Cmd {
  return textinput.Blink 
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// 1. Handle Global Keys and Logic
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil // Don't propagate resize to children yet

	case components.ActionMsg:
		// 1. Log the action for debugging
		m.logger.Printf("Action Triggered: %s (ID: %s)", msg.Action, msg.ID)
		
		// 2. Pass it to the navigator to handle view switching or commands
		res := m.navigator.ProcessAction(msg.Action, m.getContextData())
		
		if res.Command != "" {
			// This is where you'd run the actual shell command (e.g., git commit)
			return m, m.executeShellCommand(res.Command)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			// A. Pull data from all components into the Context bucket
			m.syncContext()
    
			// B. Find the focused component's action
			view, _ := m.navigator.GetActiveView()
			var action string
			for _, child := range view.Children {
				if child.IsFocusable() {
					action = child.GetAction()
					break 
				}
			}

			// C. Process the action through the Navigator
			if action != "" {
				res := m.navigator.ProcessAction(action, m.navigator.Context.Data)
				
				if res.IsUpdate {
					// It was a view swap; just return to re-render the next view
					return m, nil
				} else if res.Command != "" {
					// It was a shell command (like git status)
					return m, m.execute(res.Command)
				}
			}
		}
	}

	// 2. Standard Flow: Propagate the message down to the current view
	view, ctx := m.navigator.GetActiveView()
	ctx.Width = m.width
	ctx.Height = m.height

	updatedView, viewCmd := view.Update(msg, ctx)
	
	// 3. Persist the view state (cursor positions, text input buffers)
	m.navigator.Views[m.navigator.ActiveViewID] = updatedView

	return m, viewCmd
}

func (m model) View() string {
	view, ctx := m.navigator.GetActiveView()
	
	// Pass the width/height again just to be sure
	ctx.Width = m.width
	ctx.Height = m.height

	return view.Render(ctx)
}

// collectData gathers all GetValue() results from the current view
func collectData(v *components.View) map[string]string {
  data := make(map[string]string)
  for _, child := range v.Children {
    if child.GetID() != "" {
      data[child.GetID()] = child.GetValue()
    }
  }
  return data
}

// execute turns a command string into a Bubble Tea command
func (m model) execute(action string) tea.Cmd {
  cParts := strings.Fields(action)
  if len(cParts) > 0 {
    return tea.ExecProcess(exec.Command(cParts[0], cParts[1:]...), func(err error) tea.Msg {
      return nil
    })
  }
  return nil
}
// Simple interpolation helper inside main
func interpolate(text string, data map[string]string) string {
  for k, v := range data {
    text = strings.ReplaceAll(text, "{{."+k+"}}", v)
  }
  return text
}

func main() {
  if len(os.Args) < 2 {
    fmt.Println("Usage: tuik <config.json>")
    os.Exit(0)
  }

  configFile := os.Args[1]

  content, err := os.ReadFile(configFile)
  if err != nil {
    fmt.Printf("File error: %v\n", err)
    os.Exit(1)
  }

  // Use our new parser package
  cfg, err := parser.ParseConfig(content)
  if err != nil {
    fmt.Printf("Parse error: %v\n", err)
    os.Exit(1)
  }

  p := tea.NewProgram(initialModel(cfg))
  if _, err := p.Run(); err != nil {
    fmt.Printf("Runtime error: %v\n", err)
    os.Exit(1)
  }
}
