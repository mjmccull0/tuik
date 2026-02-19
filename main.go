package main

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type menuState struct {
	choices []item
	cursor  int
}

type item struct {
	Label string `json:"label"`
	Exec  string `json:"exec"`
	Items []item `json:"items"`
}

type model struct {
	stack    []menuState
	choices  []item
	cursor   int
	chosen   string
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc", "backspace":
			if len(m.stack) > 0 {
				// RESTORE the previous state
				lastIndex := len(m.stack) - 1
				previousState := m.stack[lastIndex]
				
				m.choices = previousState.choices
				m.cursor = previousState.cursor // Focus returns to the parent item!
				
				m.stack = m.stack[:lastIndex]
				return m, nil
			}
		case "enter":
			current := m.choices[m.cursor]
			if len(current.Items) > 0 {
				// Enter sub-menu
				m.stack = append(m.stack, menuState{
				  choices: m.choices,
					cursor: m.cursor,
			  })
				m.choices = current.Items
				m.cursor = 0
				return m, nil
			}
			// It's an executable command
			m.chosen = os.ExpandEnv(current.Exec)
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.choices)-1 { m.cursor++ }
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Mink Menu\n\n"
	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		s += fmt.Sprintf("%s%s\n", cursor, choice.Label)
	}
	return s + "\n(q to quit)\n"
}

func main() {
	// 1. Check for the file argument
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: mink <config.json>")
		os.Exit(1)
	}
	configPath := os.Args[1]

	// 2. Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var items []item
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// 3. Run the Program
	p := tea.NewProgram(model{choices: items}, tea.WithOutput(os.Stderr))
	m, err := p.Run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 4. Output selection
	finalModel := m.(model)
	if finalModel.chosen != "" {
		fmt.Print(finalModel.chosen)
	}
}
