package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Define a constant for the data exchange file
const tempOutputFile = "/tmp/tuik_exchange.tmp"

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

	// 1. CLEAR THE TEMP FILE BEFORE RUNNING
	// This prevents stale data (like "search") from being read by the next view.
	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// 2. Resolve Environment Variables
	env := os.Environ()
	for k, val := range v.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, m.ctx.Resolve(val)))
	}

	// 3. Resolve and Quote Arguments
	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		val := m.ctx.Resolve(arg)
		// Quote arguments containing spaces to prevent shell splitting
		if strings.Contains(val, " ") {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

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
	if v.Component == "gum" {
		cmdString = fmt.Sprintf("%s > %s", cmdString, tempOutputFile)
	}

	os.WriteFile("debug_cmd.log", []byte(cmdString), 0644)

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
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	var cfg TuikConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	m := &model{
		cfg:          cfg,
		activeViewID: cfg.Main,
		ctx: Context{
			Data:   make(map[string]string),
			Styles: cfg.Styles,
		},
	}

	// Use AltScreen to ensure the terminal is cleaned up after exit
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Runtime Error: %v\n", err)
	}
}
