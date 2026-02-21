package main

import (
	"encoding/json"
	"fmt"
	"os"
	// "regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles & Context ---

type StyleConfig struct {
	Color           string `json:"color"`
	BackgroundColor string `json:"background-color"`
	Bold            bool   `json:"bold"`
	Italic          bool   `json:"italic"`
	Underline       bool   `json:"underline"`
	Padding         int    `json:"padding"`
	MarginTop       int    `json:"margin-top"`
	MarginBottom    int    `json:"margin-bottom"`
}

type StyleContext struct {
	Foreground string
	Background string
}

// --- Components ---

type ListItem struct {
	Label    string `json:"label"`
	Exec     string `json:"exec"`
	Selected bool
	Color    string
}

type Component struct {
	Type     string      `json:"type"`
	Value    string      `json:"value,omitempty"`
	Style    StyleConfig `json:"style"`
	Children []Component `json:"children,omitempty"`
	// List specific
	Items       []ListItem `json:"items,omitempty"`
	MultiSelect bool       `json:"multi-select"`
}

// --- Bubble Tea Model ---

type model struct {
	root   Component
	cursor int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Find the list in the tree to get item count (simplified for PoC)
	listComp := findList(&m.root)
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if listComp != nil && m.cursor < len(listComp.Items)-1 { m.cursor++ }
		case " ":
			if listComp != nil && listComp.MultiSelect {
				listComp.Items[m.cursor].Selected = !listComp.Items[m.cursor].Selected
			}
		}
	}
	return m, nil
}

// Recursive Renderer
func (c *Component) Render(ctx StyleContext, globalCursor int) string {
	// 1. Inherit Styles (Colors cascade)
	if c.Style.Color != "" { ctx.Foreground = c.Style.Color }
	if c.Style.BackgroundColor != "" { ctx.Background = c.Style.BackgroundColor }

	var content string

	// 2. Render by Type
	switch c.Type {
	case "container", "view":
		var parts []string
		for i := range c.Children {
			parts = append(parts, c.Children[i].Render(ctx, globalCursor))
		}
		content = strings.Join(parts, "\n")

	case "text":
		content = c.Value

	case "list":
		var lines []string
		for i, item := range c.Items {
			// Gutter Logic (Stability)
			cursor := "  "
			if globalCursor == i { cursor = "> " }
			
			prefix := ""
			if c.MultiSelect {
				check := " "
				if item.Selected { check = "x" }
				prefix = fmt.Sprintf("[%s] ", check)
			}

			// Apply item specific colors
			itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ctx.Foreground))
			if item.Selected {
				itemStyle = itemStyle.Foreground(lipgloss.Color("42")) // Green
			}
			
			lines = append(lines, fmt.Sprintf("%s%s%s", cursor, prefix, itemStyle.Render(item.Label)))
		}
		content = strings.Join(lines, "\n")
	}

	// 3. Apply Layout (Containment/Padding/Margin)
	res := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ctx.Foreground)).
		Background(lipgloss.Color(ctx.Background)).
		Bold(c.Style.Bold).
		Italic(c.Style.Italic).
		Underline(c.Style.Underline).
		Padding(c.Style.Padding).
		MarginTop(c.Style.MarginTop).
		MarginBottom(c.Style.MarginBottom).
		Render(content)

	return res
}

func (m model) View() string {
	return m.root.Render(StyleContext{}, m.cursor)
}

// Helper to find the active list in the tree
func findList(c *Component) *Component {
	if c.Type == "list" { return c }
	for i := range c.Children {
		res := findList(&c.Children[i])
		if res != nil { return res }
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tuik <config.json>")
		os.Exit(1)
	}
	data, _ := os.ReadFile(os.Args[1])
	var config map[string]Component
	json.Unmarshal(data, &config)

	p := tea.NewProgram(model{root: config["main"]})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
