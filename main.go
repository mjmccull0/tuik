package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- The Shorthand Magic ---

type TextValue struct {
	Value string      `json:"value"`
	Style StyleConfig `json:"style"`
}

// UnmarshalJSON detects if 'text' is "string" or {"value": "string"}
func (t *TextValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Value = s
		return nil
	}
	type alias TextValue
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*t = TextValue(obj)
	return nil
}

// --- Structures ---

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

type Component struct {
	Type        string      `json:"type"`
	Text        TextValue   `json:"text"`
	Style       StyleConfig `json:"style"`
	Children    []Component `json:"children,omitempty"`
	Items       []ListItem  `json:"items,omitempty"`
	MultiSelect bool        `json:"multi-select"`
}

func (c *Component) UnmarshalJSON(data []byte) error {
	// Create a shadow type to avoid infinite recursion during Unmarshal
	type shadow Component
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Copy the unmarshaled data to our component
	*c = Component(s)

	// If Type is missing, infer it from the keys present
	if c.Type == "" {
		// Use a map to check for specific key existence
		var raw map[string]json.RawMessage
		json.Unmarshal(data, &raw)

		if _, ok := raw["text"]; ok {
			c.Type = "text"
		} else if _, ok := raw["items"]; ok || raw["source"] != nil {
			c.Type = "list"
		} else if _, ok := raw["children"]; ok {
			c.Type = "container"
		}
	}
	return nil
}

type ListItem struct {
	Label    string `json:"label"`
	Selected bool
}

type StyleContext struct {
	Foreground string
	Background string
}

// --- Recursive Renderer ---

func (c *Component) Render(ctx StyleContext, cursor int) string {
	// 1. Inherit/Override Colors
	if c.Style.Color != "" { ctx.Foreground = c.Style.Color }
	if c.Style.BackgroundColor != "" { ctx.Background = c.Style.BackgroundColor }

	var content string
	
	// Use 'container' as default if children exist
	kind := c.Type
	if kind == "" && len(c.Children) > 0 { kind = "container" }

	switch kind {
	case "container", "view":
		var parts []string
		for i := range c.Children {
			parts = append(parts, c.Children[i].Render(ctx, cursor))
		}
		content = strings.Join(parts, "\n")

	case "text":
		// Render the text with its local style + inherited context
		tStyle := lipgloss.NewStyle().
			Bold(c.Text.Style.Bold || c.Style.Bold).
			Underline(c.Text.Style.Underline || c.Style.Underline)
		
		if c.Text.Style.Color != "" {
			tStyle = tStyle.Foreground(lipgloss.Color(c.Text.Style.Color))
		}
		content = tStyle.Render(c.Text.Value)

	case "list":
		var lines []string
		for i, item := range c.Items {
			ptr := "  "
			if cursor == i { ptr = "> " }
			
			box := ""
			if c.MultiSelect {
				mark := " "
				if item.Selected { mark = "x" }
				box = fmt.Sprintf("[%s] ", mark)
			}
			
			label := item.Label
			if item.Selected {
				label = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(label)
			}
			lines = append(lines, fmt.Sprintf("%s%s%s", ptr, box, label))
		}
		content = strings.Join(lines, "\n")
	}

	// Apply Box Decoration (Containment)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ctx.Foreground)).
		Background(lipgloss.Color(ctx.Background)).
		Padding(c.Style.Padding).
		MarginTop(c.Style.MarginTop).
		MarginBottom(c.Style.MarginBottom).
		Render(content)
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
