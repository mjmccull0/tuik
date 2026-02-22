package components

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

func (l List) Render(ctx RenderContext) string {
	var rows []string
	for i, item := range l.Items {
		style := lipgloss.NewStyle().PaddingLeft(2)
		
		// Only show selection if the LIST itself is focused
		if ctx.IsFocused && i == l.Cursor {
			style = style.Foreground(lipgloss.Color("81")).SetString("> " + item.Text)
		} else {
			style = style.SetString("  " + item.Text)
		}
		rows = append(rows, style.Render())
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (l List) Update(msg tea.Msg) (Component, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return l, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if l.Cursor > 0 {
			l.Cursor--
		}
	case "down", "j":
		if l.Cursor < len(l.Items)-1 {
			l.Cursor++
		}
	case "enter", " ":
		// Return a special command or action string? 
		// For now, we'll let the parent handle the "Action" execution logic
		// by inspecting the active item.
	}

	return l, nil
}

func (l List) IsFocusable() bool { return true }
func (l List) GetType()    string { return "list" }
func (l List) GetID() string     { return "" }
func (l List) GetValue() string  { return "" }
func (l List) GetAction() string { 
    if l.Cursor >= 0 && l.Cursor < len(l.Items) {
        return l.Items[l.Cursor].OnPress
    }
    return ""
}
