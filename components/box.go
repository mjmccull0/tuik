package components

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Box struct {
	ID       string
	Children []Component
	Vertical bool
	Styles   StyleConfig 
}

func (b *Box) Render(ctx Context) string {
	var views []string
	for _, child := range b.Children {
		views = append(views, child.Render(ctx))
	}

	var out string
	if b.Vertical {
		out = lipgloss.JoinVertical(lipgloss.Left, views...)
	} else {
		out = lipgloss.JoinHorizontal(lipgloss.Top, views...)
	}

	// Apply styles to the container itself (padding, borders, etc.)
	return b.Styles.ToLipgloss().Render(out)
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

// Focus propagates the focus signal down to the first focusable child
func (b *Box) Focus() {
	for _, child := range b.Children {
		if child.IsFocusable() {
			child.Focus()
			break 
		}
	}
}

// Blur ensures ALL children are blurred so nothing stays pink accidentally
func (b *Box) Blur() {
	for _, child := range b.Children {
		child.Blur()
	}
}

func (b *Box) GetAction() string { return "" }
func (b *Box) GetID() string     { return b.ID }
func (b *Box) GetType() string   { return "box" }
func (b *Box) GetValue() string  { return "" }

// A Box isn't focusable itself, but it can CONTAIN focusable things.
func (b *Box) IsFocusable() bool { return false }
