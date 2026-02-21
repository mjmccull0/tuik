package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Render is the entry point for the recursive tree walk
func (c *Component) Render(ctx StyleContext, cursor int) string {
	// 1. Inheritance: Update context with local overrides
	if c.Style.Color != "" { ctx.Foreground = c.Style.Color }
	if c.Style.BackgroundColor != "" { ctx.Background = c.Style.BackgroundColor }

	var content string

	// 2. Routing: Determine which specific renderer to use
	switch c.Type {
	case "container", "view":
		var parts []string
		for i := range c.Children {
			parts = append(parts, c.Children[i].Render(ctx, cursor))
		}
		content = strings.Join(parts, "\n")

	case "text":
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

	// 3. Containment: Wrap the rendered content in the component's own box
	return c.Style.ToLipgloss().
		Foreground(lipgloss.Color(ctx.Foreground)).
		Background(lipgloss.Color(ctx.Background)).
		Render(content)
}

// GetItemCount is a helper for main.go to know the bounds of the cursor
func (c *Component) GetItemCount() int {
	if c.Type == "list" { return len(c.Items) }
	for i := range c.Children {
		if count := c.Children[i].GetItemCount(); count > 0 {
			return count
		}
	}
	return 0
}

// ToggleItem is a helper for the spacebar
func (c *Component) ToggleItem(index int) {
	if c.Type == "list" && index < len(c.Items) {
		c.Items[index].Selected = !c.Items[index].Selected
		return
	}
	for i := range c.Children {
		c.Children[i].ToggleItem(index)
	}
}
