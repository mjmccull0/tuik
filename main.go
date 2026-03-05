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


type ShellMsg struct {
    Command string
}

type RefreshMsg struct{}

func (m *model) resolveString(input string) string {
    // Create a temporary Context object to use its Resolve method
    ctx := components.Context{
        Data: m.getContextData(),
    }
    return ctx.Resolve(input)
}

// getContextData gathers all current component values into a map for the navigator
func (m *model) getContextData() map[string]string {
	m.syncContext()
	return m.navigator.Context.Data
    // data := make(map[string]string)
    // You likely have a loop here that visits every component 
    // and calls GetValue(), similar to your existing sync logic.
    // return data
}

// Define a new message type
type shellOutputMsg string

func (m *model) executeShellCommand(cmdStr string) tea.Cmd {
    return func() tea.Msg {
        // Use the navigator's wrapped execution instead of raw exec.Command
        output, err := m.navigator.execWrapped(cmdStr)
        
        if err != nil && m.logger != nil {
             m.logger.Printf("Shell Error: %v", err)
        }

        if m.logger != nil {
            m.logger.Printf("Command Run (Wrapped): %s", cmdStr)
        }

        return shellOutputMsg(string(output))
    }
}

// prepareCmd returns the *exec.Cmd without running it, useful for tea.ExecProcess
func (n *Navigator) prepareCmd(cmdStr string) *exec.Cmd {
    // This ensures 'git_sync' or other custom functions work
    finalCmd := cmdStr

    if n.Config.ShellFunctions != "" {
        finalCmd = fmt.Sprintf("source %s; %s", n.Config.ShellFunctions, cmdStr)
    }

    return exec.Command("zsh", "-c", finalCmd)
}

// executeForegroundCommand suspends the TUI to run an interactive process
func (m *model) executeForegroundCommand(cmdStr string) tea.Cmd {
	clearedCmd := fmt.Sprintf("clear -x; tput smcup; clear; { %s; }; tput rmcup", cmdStr)

	// Use our new prepareCmd to get the zsh + source wrapper
	// c := m.navigator.prepareCmd(cmdStr)
	c := m.navigator.prepareCmd(clearedCmd)
	utils.Log("SHELL EXEC: Running '%s' with functions from '%s'", 
              cmdStr, m.navigator.Config.ShellFunctions)
	
	utils.Log("SHELL EXEC: Running Wrapped Foreground Cmd: %s", cmdStr)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil && m.logger != nil {
			m.logger.Printf("Foreground Error: %v", err)
		}
		return RefreshMsg{}
	})
}

func (m *model) syncContext() {
    view, _ := m.navigator.GetActiveView()
    if m.navigator.Context.Data == nil {
        m.navigator.Context.Data = make(map[string]string)
    }
    
    // Instead of the manual loop, let the recursion do the work
    for _, child := range view.Children {
        m.pullComponentData(child)
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
		Config:       Config{Main: cfg.Main, ShellFunctions: cfg.ShellFunctions},
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

	// Only sync if we actually have a valid view to sync from
	if _, ok := nav.Views[nav.ActiveViewID]; ok {
			m.syncContext()
	} else {
			utils.Log("Warning: Initial view %s not found in config", nav.ActiveViewID)
	}
	m.syncContext()
	return m
}

func (m *model) Init() tea.Cmd {
    // 1. Get the hydration command for the initial view
    _, hydrationCmd := m.navigator.InitView(m.navigator.ActiveViewID)

    // 2. Combine it with the text input blinker
    return tea.Batch(
        textinput.Blink, 
        hydrationCmd,
    )
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        return m, nil

    case RefreshMsg:
        _, cmd := m.navigator.InitView(m.navigator.ActiveViewID)
        return m, cmd

    case ShellMsg:
        return m, m.executeForegroundCommand(msg.Command)

    case shellOutputMsg:
        m.lastOutput = string(msg)
        return m, nil

    case components.ActionMsg:
        // Remove m.syncContext() from here
        res := m.navigator.ProcessAction(msg.Action)
        if res.NextViewID != "" {
            m.navigator.ActiveViewID = res.NextViewID
            _, cmd := m.navigator.InitView(res.NextViewID)
            return m, cmd
        }
        if res.Command != "" {
            return m, m.executeForegroundCommand(res.Command)
        }

    case tea.KeyMsg:
        // Remove m.syncContext() from here
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "esc":
            m.lastOutput = ""
            return m, nil
        }
    }

    // 1. Standard Flow: Propagate the message to the components
    view, ctx := m.navigator.GetActiveView()
    ctx.Width, ctx.Height = m.width, m.height

    updatedView, viewCmd := view.Update(msg, &ctx)
    if v, ok := updatedView.(*components.View); ok {
        m.navigator.Views[m.navigator.ActiveViewID] = v
    }

    // 2. FINAL STEP: Sync the context AFTER the components have processed the message.
    // This fixes the "one key behind" issue.
    m.syncContext()

    return m, viewCmd
}

func (m *model) View() string {
    // 1. Get the current view from the navigator
    view, ctx := m.navigator.GetActiveView()
    
    // 2. Render the components defined in your JSON
    // This is the "Main Window"
    mainContent := view.Render(&ctx)

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
func (m *model) execute(action string) tea.Cmd {
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

    p := tea.NewProgram(&m, tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Runtime error: %v\n", err)
        os.Exit(1)
    }
}
