package components

import (
	"github.com/charmbracelet/lipgloss"
  "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type TextInput struct {
	ID          string
	Content     string // This is the LOCAL state
	Placeholder string
	Model       textinput.Model
}

func (t TextInput) Render(ctx RenderContext) string {
	// 1. Tell the library model whether it should show a cursor or not
	if ctx.IsFocused {
		t.Model.Focus()
	} else {
		t.Model.Blur()
	}

	// 2. Apply your custom container styling
	style := lipgloss.NewStyle().PaddingLeft(1).Width(30)
	
	if ctx.IsFocused {
		return style.Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(t.Model.View()) // Render the library view (with cursor)
	}
	
	return style.Border(lipgloss.HiddenBorder()).
		Underline(true).
		Foreground(lipgloss.Color("240")). 
		Render(t.Model.View()) // Render the blurred library view
}

func (t TextInput) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	
	// 1. Let the library handle all the complex keyboard logic (backspace, etc.)
	t.Model, cmd = t.Model.Update(msg)
	
	// 2. Sync the library's internal state to your local Content field
	t.Content = t.Model.Value()

	return t, cmd 
}

func (t TextInput) GetValue() string { return t.Content }
func (t TextInput) IsFocusable() bool { return true }
func (t TextInput) GetID() string     { return t.ID }
func (t TextInput) GetAction() string { return "" }
func (t TextInput) GetType() string { return "text-input" }
