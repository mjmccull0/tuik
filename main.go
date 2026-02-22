package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"tuik/components"
	"tuik/parser"
)

type model struct {
	registry   map[string]*components.View
	active     string
	focusIndex int
}

func initialModel(cfg components.Config) model {
	m := model{
		registry: cfg.Views,
		active:   cfg.Main,
	}

	// Helper to find initial focus
	if view, ok := m.registry[m.active]; ok {
		for i, child := range view.Children {
			if child.IsFocusable() {
				m.focusIndex = i
				break
			}
		}
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink 
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	view := m.registry[m.active]
	if view == nil || len(view.Children) == 0 {
		return m, nil
	}

	activeComp := view.Children[m.focusIndex]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			// Navigation logic
			for i := 0; i < len(view.Children); i++ {
				m.focusIndex = (m.focusIndex + 1) % len(view.Children)
				if view.Children[m.focusIndex].IsFocusable() {
					break
				}
			}
			return m, nil

		case "enter", " ":
			// 1. Gather action string from component
			actionStr := activeComp.GetAction()

			// 2. Collect state for interpolation
			localData := make(map[string]string)
			for _, child := range view.Children {
				if id := child.GetID(); id != "" {
					localData[id] = child.GetValue()
				}
			}

			// 3. Interpolate using a local helper or move to parser package
			action := interpolate(actionStr, localData)
			if action == "" {
				return m, nil
			}

			// 4. View Swap Check
			if _, exists := m.registry[action]; exists {
				m.active = action
				// Reset focus for the new view
				for i, child := range m.registry[m.active].Children {
					if child.IsFocusable() {
						m.focusIndex = i
						break
					}
				}
				return m, nil
			}

			// 5. Shell Execution
			cParts := strings.Fields(action)
			if len(cParts) > 0 {
				return m, tea.ExecProcess(exec.Command(cParts[0], cParts[1:]...), func(err error) tea.Msg {
					return nil
				})
			}
		}

		// DELEGATION: Pass the message to the active component
		newComp, cmd := activeComp.Update(msg)
		view.Children[m.focusIndex] = newComp
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	view := m.registry[m.active]
	if view == nil {
		return "Error: View not found"
	}

	var rendered []string
	for i, child := range view.Children {
		// SAFETY CHECK: If the parser returned nil for a component
		if child == nil {
			rendered = append(rendered, "[Invalid Component]")
			continue
		}

		ctx := components.RenderContext{
			IsFocused: (i == m.focusIndex),
		}
		rendered = append(rendered, child.Render(ctx))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
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
