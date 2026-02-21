package components

import (
	"fmt"
	"strings"
	"github.com/charmbracelet/lipgloss"
)

// Define this here so the Engine can pass color data in
type StyleContext struct {
	Foreground string
	Background string
}

type ListItem struct {
	Label    string `json:"label"`
	Selected bool
}

func RenderList(items []ListItem, multi bool, ctx StyleContext, cursor int) string {
	var lines []string
	for i, item := range items {
		ptr := "  "
		if cursor == i {
			ptr = "> "
		}

		box := ""
		if multi {
			mark := " "
			if item.Selected {
				mark = "x"
			}
			box = fmt.Sprintf("[%s] ", mark)
		}

		// Use the inherited context colors for the base style
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(ctx.Foreground))
		
		label := item.Label
		if item.Selected {
			// Highlight selected items in green
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(label)
		}
		
		lines = append(lines, fmt.Sprintf("%s%s%s", ptr, box, style.Render(label)))
	}
	return strings.Join(lines, "\n")
}
