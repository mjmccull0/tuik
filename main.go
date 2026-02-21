package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"fmt"
  "github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"tuik/ui"
)

type model struct {
	registry map[string]*ui.Component
	focusIndex int               // Track which child is active (0, 1, 2...)
	active   string // The ID of the current view
	cursor   int
	store    map[string]string
	focus    string
}

func initialModel(cfg ui.Config) model {
	return model{
		registry: cfg.Views,
		active:   cfg.Main,
		cursor:   0,
		store:    make(map[string]string), // CRITICAL: Initialize the map
	}
}

func interpolate(action string, store map[string]string) string {
    for key, val := range store {
        action = strings.ReplaceAll(action, "{{."+key+"}}", val)
    }
    return action
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	view := m.registry[m.active]
	if view == nil || len(view.Children) == 0 {
		return m, nil
	}

	activeComp := &view.Children[m.focusIndex]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// --- SECTION 1: TEXT INPUT HANDLING ---
		// Priority: Capture typing before navigation
		if activeComp.GetType() == "text-input" {
			switch msg.String() {
			case "tab", "shift+tab", "enter", "esc":
				// Exit typing mode: continue to navigation logic
			case "backspace":
				curr := m.store[activeComp.ID]
				if len(curr) > 0 {
					m.store[activeComp.ID] = curr[:len(curr)-1]
				}
				return m, nil
			default:
				// Capture letters, symbols, and spaces
				if len(msg.String()) == 1 {
					m.store[activeComp.ID] += msg.String()
				}
				return m, nil
			}
		}

		// --- SECTION 2: GLOBAL NAVIGATION ---
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			// Discovery loop: find next focusable item
			for i := 0; i < len(view.Children); i++ {
				m.focusIndex = (m.focusIndex + 1) % len(view.Children)
				if view.Children[m.focusIndex].IsFocusable() {
					break
				}
			}
			m.cursor = 0
			return m, nil

		case "up", "k":
			if activeComp.GetType() == "list" && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if activeComp.GetType() == "list" && m.cursor < len(activeComp.Items)-1 {
				m.cursor++
			}
			return m, nil

		case "enter", " ":
			// Resolve the action string and inject variables
			action := interpolate(activeComp.GetActionAt(m.cursor), m.store)
			if action == "" {
				return m, nil
			}

			// View Swap Logic
			if _, isView := m.registry[action]; isView {
				m.active = action
				m.cursor = 0
				m.focusIndex = 0
				// Ensure new view starts on a focusable item
				for i := 0; i < len(m.registry[m.active].Children); i++ {
					if m.registry[m.active].Children[m.focusIndex].IsFocusable() {
						break
					}
					m.focusIndex = (m.focusIndex + 1) % len(m.registry[m.active].Children)
				}
				return m, nil
			}

			// Shell Execution Logic
			c := strings.Fields(action)
			if len(c) > 0 {
				return m, tea.ExecProcess(exec.Command(c[0], c[1:]...), func(err error) tea.Msg {
					return nil
				})
			}
		}
	}
	return m, nil
}

func (m model) View() string {
    view := m.registry[m.active]
    var rendered []string
    for i := range view.Children {
        // Pass 'true' only if this child is the currently focused index
        isFocused := (i == m.focusIndex)
        rendered = append(rendered, view.Children[i].Render(ui.StyleContext{},m.cursor, isFocused, m.store))

    }
    return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

func main() {
    // 1. Load your JSON config
    content, _ := os.ReadFile("tuik.json")
    var cfg ui.Config
    json.Unmarshal(content, &cfg)

    // 2. Initialize the model with the store allocated
    p := tea.NewProgram(initialModel(cfg))
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
