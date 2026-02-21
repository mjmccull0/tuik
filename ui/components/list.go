package components

import (
	"fmt"
	"strings"
)

// Define this here so the Engine can pass color data in
type StyleContext struct {
	Foreground string
	Background string
}

type ListItem struct {
	Text     TextValue  `json:"text"`
	OnPress  string     `json:"on-press"`
	Selected bool
}

func RenderList(items []ListItem, multi bool, ctx StyleContext, cursor int) string {
	var lines []string
	for i, item := range items {
		ptr := "  " // Two spaces
		if cursor == i {
			ptr = "> "
		}

		// content already contains the lipgloss-rendered string from RenderText
		content := RenderText(item.Text)
		
		// Use only two placeholders to match your variables
		lines = append(lines, fmt.Sprintf("%s%s", ptr, content))
	}
	return strings.Join(lines, "\n")
}
