package main

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	// Replace 'your-project-name' with the name from your go.mod file
	"tuik/ui" 
)

type model struct {
	root   ui.Component
	cursor int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Get item count from the UI tree to bound the cursor
	itemCount := m.root.GetItemCount()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < itemCount-1 {
				m.cursor++
			}

		case " ":
			// Use the helper we wrote in ui/renderers.go
			m.root.ToggleItem(m.cursor)

		case "enter":
			// Future: Logic for 'exec' commands will go here
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	// Start the recursive render with an empty context
	// Colors will inherit from the 'main' component down
	initialCtx := ui.StyleContext{}
	return m.root.Render(initialCtx, m.cursor)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tuik <config.json>")
		os.Exit(1)
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Could not read file: %v\n", err)
		os.Exit(1)
	}

	// Unmarshal into a map to find the "main" entry point
	var config map[string]ui.Component
	if err := json.Unmarshal(content, &config); err != nil {
		fmt.Printf("JSON Error: %v\n", err)
		os.Exit(1)
	}

	mainComp, ok := config["main"]
	if !ok {
		fmt.Println("Error: JSON must contain a 'main' object.")
		os.Exit(1)
	}

	p := tea.NewProgram(model{root: mainComp})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, it failed: %v", err)
		os.Exit(1)
	}
}
