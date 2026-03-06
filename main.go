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

	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	env := os.Environ()
	for k, val := range v.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, m.ctx.Resolve(val)))
	}

	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		val := m.ctx.Resolve(arg)
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



func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processFinishedMsg:
		if msg.err != nil {
			if m.activeViewID == m.cfg.Main {
				return m, tea.Quit
			}
			m.activeViewID = m.cfg.Main
			return m, m.runActiveView()
		}

		v := m.cfg.Views[m.activeViewID]

		// 1. Capture output into state
		if out, err := os.ReadFile(tempOutputFile); err == nil && len(out) > 0 {
			val := strings.TrimSpace(string(out))
			m.ctx.Data["last_output"] = val
			if v.SetState != "" {
				m.ctx.Data[v.SetState] = val
			}
			_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
		}

		// 2. Resolve Navigation and Transition State
		var nextTarget string
		switch outcome := v.OnSuccess.(type) {
		case string:
			nextTarget = outcome

		case map[string]interface{}:
			var rawTarget interface{}

			// Check if this is a direct Handoff (has "target")
			if target, ok := outcome["target"].(string); ok {
				nextTarget = target
				rawTarget = outcome
			} else {
				// Otherwise, treat as a Branching Map (Menu)
				choice := m.ctx.Data["last_output"]
				if branch, ok := outcome[choice]; ok {
					rawTarget = branch
				}
			}

			// Handle the target (whether it came from a Handoff or a Menu Branch)
			switch t := rawTarget.(type) {
			case string:
				nextTarget = t
			case map[string]interface{}:
				if targetName, ok := t["target"].(string); ok {
					nextTarget = targetName
				}
				// Apply nested set_state for this transition
				if newState, ok := t["set_state"].(map[string]interface{}); ok {
					for k, val := range newState {
						m.ctx.Data[k] = m.ctx.Resolve(fmt.Sprintf("%v", val))
					}
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
		fmt.Printf("Error: %v\n", err)
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

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
