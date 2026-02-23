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
	lastOutput string
}


func (m model) resolveString(input string) string {
    // Create a temporary Context object to use its Resolve method
    ctx := components.Context{
        Data: m.getContextData(),
    }
    return ctx.Resolve(input)
}

// getContextData gathers all current component values into a map for the navigator
func (m model) getContextData() map[string]string {
	m.syncContext()
	return m.navigator.Context.Data
    // data := make(map[string]string)
    // You likely have a loop here that visits every component 
    // and calls GetValue(), similar to your existing sync logic.
    // return data
}

// Define a new message type
type shellOutputMsg string

func (m model) executeShellCommand(cmdStr string) tea.Cmd {
    return func() tea.Msg {
        cmd := exec.Command("sh", "-c", cmdStr)
        output, _ := cmd.CombinedOutput()
        
        if m.logger != nil {
            m.logger.Printf("Command Run: %s", cmdStr)
        }

        // Return the output so Update() can catch it
        return shellOutputMsg(string(output))
    }
}

// executeForegroundCommand suspends the TUI to run an interactive process
func (m model) executeForegroundCommand(cmdStr string) tea.Cmd {
	// We use tea.ExecProcess to handle the terminal hand-off
	// It takes an *exec.Cmd and a function to return a msg when finished
	c := exec.Command("sh", "-c", cmdStr)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			if m.logger != nil {
				m.logger.Printf("Foreground Error: %v", err)
			}
		}
		return nil // Return to the TUI normally
	})
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
	viewMap := make(map[string]*components.View)
	for k, v := range cfg.Views {
		viewMap[k] = v // Convert pointer to value if needed
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
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        return m, nil

		case shellOutputMsg:
			// Store it in the model so View() can see it
			m.lastOutput = string(msg)
			return m, nil

		case components.ActionMsg:
			// 1. Get current data
			ctxData := m.getContextData()
			
			// 2. Resolve the action string (this swaps {{.selected_branch}} for "main")
			resolvedAction := m.navigator.Context.Resolve(msg.Action)
			
			// 3. Process the resolved action
			res := m.navigator.ProcessAction(resolvedAction, ctxData)
			
			if res.NextViewID != "" {
				m.lastOutput = ""
				return m, nil
			}
			
			if res.Command != "" {
				// Use the same prefix logic here
				if strings.HasPrefix(res.Command, "shell:") {
					trimmedCmd := strings.TrimPrefix(res.Command, "shell:")

					return m, m.executeForegroundCommand(trimmedCmd)
				}
				return m, m.executeShellCommand(res.Command)
			}

    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
			  case "esc":
		        m.lastOutput = ""
			      return m, nil
        }
    }

    // Standard Flow: Only propagate to the view if it wasn't a global key
    view, ctx := m.navigator.GetActiveView()
    ctx.Width, ctx.Height = m.width, m.height

    updatedView, viewCmd := view.Update(msg, ctx)
	  if v, ok := updatedView.(*components.View); ok {
			m.navigator.Views[m.navigator.ActiveViewID] = v
	  }

    return m, viewCmd
}

func (m model) View() string {
    // 1. Get the current view from the navigator
    view, ctx := m.navigator.GetActiveView()
    
    // 2. Render the components defined in your JSON
    // This is the "Main Window"
    mainContent := view.Render(ctx)

    // 3. If a command was run, append the result to the bottom
    if m.lastOutput != "" {
        // Create a style for the status bar
        statusStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("86")). // Cyan-ish
            Border(lipgloss.RoundedBorder(), true, false, false, false). // Top border only
            BorderForeground(lipgloss.Color("240")).
            Padding(1, 0).
            Width(m.width)

        // Trim output so it doesn't push the UI off-screen
        lines := strings.Split(m.lastOutput, "\n")
        if len(lines) > 8 {
            lines = append(lines[:8], "... (truncated)")
        }
        displayOutput := strings.Join(lines, "\n")

        // Combine the JSON UI and the Shell Output
        return lipgloss.JoinVertical(
            lipgloss.Left, 
            mainContent, 
            statusStyle.Render("  Last Command Output:\n"+displayOutput),
        )
    }

    return mainContent
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

    // --- ADD THIS BLOCK ---
    // 1. Setup Logging to a file so it doesn't mess up the TUI
    f, err := tea.LogToFile("tuik.log", "debug")
    if err != nil {
        fmt.Printf("Could not open log file: %v\n", err)
        os.Exit(1)
    }
    defer f.Close()
    
    // Create the standard logger instance
    logger := log.New(f, "", log.LstdFlags)
    // -----------------------

    configFile := os.Args[1]
    content, err := os.ReadFile(configFile)
    if err != nil {
        fmt.Printf("File error: %v\n", err)
        os.Exit(1)
    }

    cfg, err := parser.ParseConfig(content)
    if err != nil {
        fmt.Printf("Parse error: %v\n", err)
        os.Exit(1)
    }

    // 2. Pass the logger into the initial model
    m := initialModel(cfg)
    m.logger = logger 

    p := tea.NewProgram(m)
    if _, err := p.Run(); err != nil {
        fmt.Printf("Runtime error: %v\n", err)
        os.Exit(1)
    }
}
