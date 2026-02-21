package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"tuik/ui"
)

type model struct {
	registry map[string]*ui.Component
	active   string // The ID of the current view
	cursor   int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	currentView := m.registry[m.active]
	if currentView == nil { return m, nil }

	itemCount := currentView.GetItemCount()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < itemCount-1 { m.cursor++ }

		case "enter", " ":
				action := currentView.GetActionAt(m.cursor)
				if action == "" {
						return m, nil
				}

				// 1. Check if the string matches a View ID in our Registry
				if _, isView := m.registry[action]; isView {
						m.active = action
						m.cursor = 0
						return m, nil
				}

				// 2. Otherwise, treat it as a Shell Command
				c := strings.Fields(action)
				if len(c) > 0 {
						return m, tea.ExecProcess(exec.Command(c[0], c[1:]...), func(err error) tea.Msg {
								// We can return a custom message here to handle errors
								return nil 
						})
				}
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.registry[m.active].Render(ui.StyleContext{}, m.cursor)
}

func main() {
	content, _ := os.ReadFile("tuik.json")
	var config ui.Config
	json.Unmarshal(content, &config)

	p := tea.NewProgram(model{
		registry: config.Views,
		active:   config.Main,
	})
	p.Run()
}
