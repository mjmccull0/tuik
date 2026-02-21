package ui

import (
	"encoding/json"

	"github.com/charmbracelet/lipgloss"
	"tuik/ui/components"
)

// ViewRegistry holds all available views by ID
type ViewRegistry map[string]Component

type Config struct {
	Main  string                `json:"main"`
	Views map[string]*Component `json:"views"`
}

type Component struct {
	Type        string               `json:"type"`
	ID          string               `json:"id,omitempty"`
	Text        components.TextValue `json:"text"`
	Style       StyleConfig          `json:"style"`
	Children    []Component          `json:"children,omitempty"`
	Items       []components.ListItem `json:"items,omitempty"`
	MultiSelect bool                 `json:"multi-select"`
	OnPress     string               `json:"on-press"`
}

// 1. Fixed the missing GetType method
func (c *Component) GetType() string {
	if c.Type != "" {
		return c.Type
	}
	if len(c.Children) > 0 {
		return "container"
	}
	if c.Text.Value != "" {
		return "text"
	}
	if len(c.Items) > 0 {
		return "list"
	}
	return "container"
}

func (c *Component) GetActionAt(cursor int) string {
	// If this IS the list, return the action at the cursor
	if c.GetType() == "list" {
		if cursor >= 0 && cursor < len(c.Items) {
			return c.Items[cursor].OnPress
		}
		return ""
	}

	// If it's a container, we have to find which child holds the cursor
	// For now, let's simplify: check children recursively
	for _, child := range c.Children {
		if action := child.GetActionAt(cursor); action != "" {
			return action
		}
	}
	
	// Fallback to the component's own action (for buttons)
	return c.OnPress
}

func (c *Component) Render(ctx StyleContext, cursor int) string {
	// 1. Inheritance
	if c.Style.Color != "" {
		ctx.Foreground = c.Style.Color
	}
	if c.Style.BackgroundColor != "" {
		ctx.Background = c.Style.BackgroundColor
	}

	var content string

	// 2. Delegate to specific component logic
	switch c.GetType() {
	case "container":
		// Handle the recursion here to keep components package clean
		var rendered []string
		for i := range c.Children {
			rendered = append(rendered, c.Children[i].Render(ctx, cursor))
		}
		content = components.RenderContainer(rendered)

	case "text":
		// Matches the signature: RenderText(t components.TextValue)
		content = components.RenderText(c.Text)

	case "list":
		// Matches the signature: RenderList(items, multi, ctx, cursor)
		// Convert our local StyleContext to components.StyleContext
		compCtx := components.StyleContext{
			Foreground: ctx.Foreground,
			Background: ctx.Background,
		}
		content = components.RenderList(c.Items, c.MultiSelect, compCtx, cursor)
	}

	// 3. Final Wrap
	return c.Style.ToLipgloss().
		Foreground(lipgloss.Color(ctx.Foreground)).
		Background(lipgloss.Color(ctx.Background)).
		Render(content)
}

// UnmarshalJSON remains here as part of the Engine
func (c *Component) UnmarshalJSON(data []byte) error {
	type shadow Component
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = Component(s)

	if c.Type == "" {
		var raw map[string]json.RawMessage
		json.Unmarshal(data, &raw)
		if _, ok := raw["text"]; ok {
			c.Type = "text"
		} else if _, ok := raw["items"]; ok {
			c.Type = "list"
		} else if _, ok := raw["children"]; ok {
			c.Type = "container"
		}
	}
	return nil
}

// Helper methods for main.go
func (c *Component) GetItemCount() int {
	if c.GetType() == "list" {
		return len(c.Items)
	}
	for i := range c.Children {
		if count := c.Children[i].GetItemCount(); count > 0 {
			return count
		}
	}
	return 0
}

func (c *Component) ToggleItem(index int) {
	if c.GetType() == "list" && index < len(c.Items) {
		c.Items[index].Selected = !c.Items[index].Selected
		return
	}
	for i := range c.Children {
		c.Children[i].ToggleItem(index)
	}
}
