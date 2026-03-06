package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const tempOutputFile = "/tmp/tuik_exchange.tmp"

type model struct {
	cfg          TuikConfig
	activeViewID string
	nav          *Navigator // The "Brain"
}

type processFinishedMsg struct {
	err error
}

func (m *model) Init() tea.Cmd {
	return m.runActiveView()
}

// runActiveView handles the OS-level execution
func (m *model) runActiveView() tea.Cmd {
	v, ok := m.cfg.Views[m.activeViewID]
	if !ok {
		return tea.Quit
	}

	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// Use the Navigator to resolve variables for Env and Args
	env := os.Environ()
	for k, val := range v.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, m.nav.Resolve(val)))
	}

	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		val := m.nav.Resolve(arg)
		if strings.Contains(val, " ") {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

	cmdString := v.Component + " " + strings.Join(resolvedArgs, " ")
	if v.Component == "gum" {
		cmdString = fmt.Sprintf("%s > %s", cmdString, tempOutputFile)
	}

	c := exec.Command("sh", "-c", cmdString)
	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return processFinishedMsg{err: err}
	})
}

// capture handles the transition after a process exits
func (m *model) capture(msg processFinishedMsg) (tea.Model, tea.Cmd) {
	// 1. Handle Errors (Canceled or Crashed)
	if msg.err != nil {
		if m.activeViewID == m.cfg.Main {
			return m, tea.Quit
		}
		m.activeViewID = m.cfg.Main
		return m, m.runActiveView()
	}

	view := m.cfg.Views[m.activeViewID]

	// 2. Capture Output into Navigator State
	if out, err := os.ReadFile(tempOutputFile); err == nil {
		val := strings.TrimSpace(string(out))
		m.nav.Set("last_output", val)
		if view.SetState != "" {
			m.nav.Set(view.SetState, val)
		}
		_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
	}

	// 3. Let Navigator decide where to go
	nextTarget := m.nav.DetermineNext(view.OnSuccess)
	return m.navigate(nextTarget)
}

// navigate updates the model's current view pointer
func (m *model) navigate(target string) (tea.Model, tea.Cmd) {
	if target == "exit" || target == "" {
		return m, tea.Quit
	}

	if strings.HasPrefix(target, "view:") {
		m.activeViewID = strings.TrimPrefix(target, "view:")
		return m, m.runActiveView()
	}

	return m, tea.Quit
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processFinishedMsg:
		return m.capture(msg)
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
		fmt.Printf("Error: %v\n", err)
		return
	}

	var cfg TuikConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Initialize the Brain
	nav := &Navigator{
		Config: cfg,
		Data:   make(map[string]string),
		Styles: cfg.Styles,
	}

	// Initialize the Body
	m := &model{
		cfg:          cfg,
		activeViewID: cfg.Main,
		nav:          nav,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
