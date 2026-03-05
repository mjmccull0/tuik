package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

<<<<<<< Updated upstream
// Define a constant for the data exchange file
const tempOutputFile = "/tmp/tuik_exchange.tmp"
=======
const tempOutputFile = "./tuik_exchange.tmp"
>>>>>>> Stashed changes

type model struct {
	cfg          TuikConfig
	ctx          Context
	activeViewID string
}

type processFinishedMsg struct {
	err error
}

func (m *model) Init() tea.Cmd {
	return m.runActiveView()
}

func (m *model) runActiveView() tea.Cmd {
	v, ok := m.cfg.Views[m.activeViewID]
	if !ok {
		return tea.Quit
	}

<<<<<<< Updated upstream
	// 1. CLEAR THE TEMP FILE BEFORE RUNNING
	// This prevents stale data (like "search") from being read by the next view.
	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// 2. Resolve Environment Variables
=======
	// Truncate exchange file so stale data isn't read
	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// 1. Resolve Environment Variables
>>>>>>> Stashed changes
	env := os.Environ()
	for k, val := range v.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, m.ctx.Resolve(val)))
	}

	// 3. Resolve and Quote Arguments
	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		val := m.ctx.Resolve(arg)
<<<<<<< Updated upstream
		// Quote arguments containing spaces to prevent shell splitting
=======
>>>>>>> Stashed changes
		if strings.Contains(val, " ") {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

<<<<<<< Updated upstream
	// 4. Construct Command
	cmdString := v.Component + " " + strings.Join(resolvedArgs, " ")

	// 5. Data Capture Logic
	// Redirect stdout to temp file for components that produce values.
	// We only do this for gum commands we want to "save" from.
	// if v.Component == "gum" && (m.activeViewID == "start" || m.activeViewID == "pick_file") {
	// 	cmdString = fmt.Sprintf("%s > %s", cmdString, tempOutputFile)
	// }

	// 5. Data Capture Logic
	// ONLY redirect to temp file for 'gum' components.
	// Neovim must have total control of Stdout to open files correctly.
=======
	// 3. Prepare Command
	cmdString := v.Component + " " + strings.Join(resolvedArgs, " ")

	// 4. Data Capture for gum
>>>>>>> Stashed changes
	if v.Component == "gum" {
		cmdString = fmt.Sprintf("%s > %s", cmdString, tempOutputFile)
	}

<<<<<<< Updated upstream
	os.WriteFile("debug_cmd.log", []byte(cmdString), 0644)

=======
>>>>>>> Stashed changes
	c := exec.Command("sh", "-c", cmdString)
	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return processFinishedMsg{err: err}
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processFinishedMsg:
<<<<<<< Updated upstream
		// 1. If the process was interrupted (Ctrl+C), exit gracefully
		if msg.err != nil {
			return m, tea.Quit
		}

		// 2. Read the captured output from our temp file
		if out, err := os.ReadFile(tempOutputFile); err == nil && len(out) > 0 {
			m.ctx.Data["last_output"] = strings.TrimSpace(string(out))
			// Truncate file so the NEXT view starts fresh
			_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
			// os.Remove(tempOutputFile) // Clean up
		}

		// 3. Handle Navigation
		v := m.cfg.Views[m.activeViewID]
		if v.OnSuccess == "exit" {
			return m, tea.Quit
		}
		
		if strings.HasPrefix(v.OnSuccess, "view:") {
			m.activeViewID = strings.TrimPrefix(v.OnSuccess, "view:")
			return m, m.runActiveView()
		}
		
=======
		// If user escapes/cancels, go back to main menu. 
		// If we are ALREADY at main, then just quit.
		if msg.err != nil {
			if m.activeViewID == m.cfg.Main {
				return m, tea.Quit
			}
			m.activeViewID = m.cfg.Main
			return m, m.runActiveView()
		}

		// Read the captured output
		if out, err := os.ReadFile(tempOutputFile); err == nil && len(out) > 0 {
			val := strings.TrimSpace(string(out))
			m.ctx.Data["last_output"] = val
			
			// NEW: If the view defines a specific key, save it there too
			v := m.cfg.Views[m.activeViewID]
			if v.SetState != "" {
				m.ctx.Data[v.SetState] = val
			}

			_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
		}

		v := m.cfg.Views[m.activeViewID]
		var nextTarget string

		// Handle Branching Logic
		switch outcome := v.OnSuccess.(type) {
		case string:
			nextTarget = outcome
		case map[string]interface{}:
			// Check if it's a Transition Object (with 'target')
			if target, ok := outcome["target"].(string); ok {
				nextTarget = target
				// Apply transition-level set_state
				if newState, ok := outcome["set_state"].(map[string]interface{}); ok {
					for k, val := range newState {
						m.ctx.Data[k] = m.ctx.Resolve(fmt.Sprintf("%v", val))
					}
				}
			} else {
				// It's a Menu Map (Branching)
				choice := m.ctx.Data["last_output"]
				if t, ok := outcome[choice].(string); ok {
					nextTarget = t
				}
			}
		}

		if nextTarget == "exit" || nextTarget == "" {
			return m, tea.Quit
		}

		if strings.HasPrefix(nextTarget, "view:") {
			m.activeViewID = strings.TrimPrefix(nextTarget, "view:")
			return m, m.runActiveView()
		}

>>>>>>> Stashed changes
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string { return "" }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tuik <config.json>")
		return
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
<<<<<<< Updated upstream
		fmt.Printf("Error reading config: %v\n", err)
=======
		fmt.Printf("Error: %v\n", err)
>>>>>>> Stashed changes
		return
	}

	var cfg TuikConfig
<<<<<<< Updated upstream
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
=======
	_ = json.Unmarshal(data, &cfg)
>>>>>>> Stashed changes

	m := &model{
		cfg:          cfg,
		activeViewID: cfg.Main,
		ctx: Context{
			Data:   make(map[string]string),
			Styles: cfg.Styles,
		},
	}

<<<<<<< Updated upstream
	// Use AltScreen to ensure the terminal is cleaned up after exit
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Runtime Error: %v\n", err)
=======
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
>>>>>>> Stashed changes
	}
}
