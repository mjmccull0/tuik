package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbletea"
)

type Box struct {
	ID       string
	Children []Component
	Vertical bool
	Styles   StyleConfig // Ensure this matches the field the parser uses
}

func (b Box) Render(ctx Context) string {
	var views []string
	for _, child := range b.Children {
		// Passing the bucket down
		views = append(views, child.Render(ctx))
	}

	if b.Vertical {
		return lipgloss.JoinVertical(lipgloss.Left, views...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

func (b *Box) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	for i, child := range b.Children {
		newComp, cmd := child.Update(msg, ctx)
		b.Children[i] = newComp
		cmds = append(cmds, cmd)
	}
	return b, tea.Batch(cmds...)
}

func (b Box) Blur()             {}
func (b *Box) Focus() {
	for _, child := range b.Children {
		if child.IsFocusable() {
			child.Focus()
			// We usually only focus the first focusable element 
			// in a container by default.
			break 
		}
	}
}
func (b Box) GetAction() string { return "" }
func (b Box) GetID() string     { return b.ID }
func (b Box) GetType() string { return "box" }
func (b Box) GetValue() string { return "" }
func (b Box) IsFocusable() bool { return false }
