package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"flag"
)

// --- Recursive Layout Schema ---

type Config struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	WidthPC   int      `yaml:"width_pc"`
	HeightPC  int      `yaml:"height_pc"`
	Direction string   `yaml:"direction"` // "vertical" or "horizontal"
	Panes     []Config `yaml:"panes"`     // Nested panes
}

// --- TUI Model ---

type pane struct {
	model textarea.Model
	conf  Config
}

type model struct {
	rootConf   Config
	flatPanes  []*pane // Linear list for easy Tabbing
	focusIndex int
	width      int
	height     int
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab":
			m.flatPanes[m.focusIndex].model.Blur()
			m.focusIndex = (m.focusIndex + 1) % len(m.flatPanes)
			return m, m.flatPanes[m.focusIndex].model.Focus()
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	var cmd tea.Cmd
	m.flatPanes[m.focusIndex].model, cmd = m.flatPanes[m.focusIndex].model.Update(msg)
	return m, cmd
}

// renderRecursive handles the math of splitting the screen
func (m *model) renderRecursive(conf Config, w, h int) string {
	if len(conf.Panes) == 0 {
		// Terminal node: find the actual textarea
		var target *pane
		for _, p := range m.flatPanes {
			if p.conf.ID == conf.ID {
				target = p
				break
			}
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(w - 2).
			Height(h - 3)

		if m.flatPanes[m.focusIndex].conf.ID == conf.ID {
			style = style.Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("205"))
		}

		target.model.SetWidth(w - 4)
		target.model.SetHeight(h - 4)

		title := lipgloss.NewStyle().Bold(true).Render(" " + conf.Title + " ")
		return title + "\n" + style.Render(target.model.View())
	}

	// Branch node: split children
	var children []string
	for _, child := range conf.Panes {
		childW := w
		childH := h
		if conf.Direction == "horizontal" {
			childW = (w * child.WidthPC) / 100
		} else {
			childH = (h * child.HeightPC) / 100
		}
		children = append(children, m.renderRecursive(child, childW, childH))
	}

	if conf.Direction == "horizontal" {
		return lipgloss.JoinHorizontal(lipgloss.Top, children...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, children...)
}

func (m *model) View() string {
	if m.width == 0 {
		return "Calculating layout..."
	}
	return m.renderRecursive(m.rootConf, m.width, m.height)
}

func main() {
	// 1. Define the flag
	configPath := flag.String("config", "", "Path to the layout YAML file")
	flag.Parse()

	// 2. Validate that a path was provided
	if *configPath == "" {
		fmt.Println("Usage: tuik-window -config <path-to-yaml>")
		os.Exit(1)
	}

	// 3. Read the specific file provided by the user
	file, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("Error: Could not read config file at %s\n%v\n", *configPath, err)
		os.Exit(1)
	}

	var root Config
	if err := yaml.Unmarshal(file, &root); err != nil {
		fmt.Printf("Error: Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	// 4. Flatten for focus management
	var flat []*pane
	var flatten func(Config)
	flatten = func(c Config) {
		// Only create a textarea for "leaf" nodes (panes with no children)
		if len(c.Panes) == 0 {
			ta := textarea.New()
			ta.Placeholder = "ID: " + c.ID
			// Ensure the textarea is focused if it's the first one
			if len(flat) == 0 {
				ta.Focus()
			}
			flat = append(flat, &pane{model: ta, conf: c})
		}
		for _, child := range c.Panes {
			flatten(child)
		}
	}
	flatten(root)

	// Safety check in case the YAML is empty
	if len(flat) == 0 {
		fmt.Println("Error: Config must contain at least one leaf pane.")
		os.Exit(1)
	}

	m := &model{rootConf: root, flatPanes: flat}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
	}
}
