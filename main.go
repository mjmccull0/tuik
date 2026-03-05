package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	cfg          TuikConfig
	ctx          Context
	activeViewID string
}

type processFinishedMsg struct {
	stdout string
	err    error
}

func (m *model) Init() tea.Cmd {
	return m.runActiveView()
}

func (m *model) runActiveView() tea.Cmd {
	v, ok := m.cfg.Views[m.activeViewID]
	if !ok {
		return tea.Quit
	}

	// 1. Resolve Environment Variables
	env := os.Environ()
	for k, val := range v.Env {
		resolvedVal := m.ctx.Resolve(val)
		env = append(env, fmt.Sprintf("%s=%s", k, resolvedVal))
	}

	// 2. Resolve Arguments
	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		resolvedArgs[i] = m.ctx.Resolve(arg)
	}

	// 3. Prepare Command
	c := exec.Command(v.Component, resolvedArgs...)
	c.Env = env

	// Capture stdout so we can pass data to the next view
	var b bytes.Buffer
	c.Stdout = &b

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return processFinishedMsg{stdout: b.String(), err: err}
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processFinishedMsg:
		// Save result to context
		m.ctx.Data["last_output"] = strings.TrimSpace(msg.stdout)

		v := m.cfg.Views[m.activeViewID]
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

	data, _ := os.ReadFile(os.Args[1])
	var cfg TuikConfig
	json.Unmarshal(data, &cfg)

	m := &model{
		cfg:          cfg,
		activeViewID: cfg.Main,
		ctx: Context{
			Data:   make(map[string]string),
			Styles: cfg.Styles,
		},
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
