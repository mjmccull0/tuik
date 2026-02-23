package components

import (
	"github.com/charmbracelet/lipgloss"
  "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	//"fmt"
)

type TextInput struct {
	ID          string
	Content     string // This is the LOCAL state
	Placeholder string
	Model       textinput.Model
}

func (t TextInput) Render(ctx Context) string {
	style := lipgloss.NewStyle().PaddingLeft(1).Width(30)
	
	return style.Border(lipgloss.HiddenBorder()).
		Underline(true).
		Foreground(lipgloss.Color("240")). 
		Render(t.Model.View())
}

func (t *TextInput) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
	// if k, ok := msg.(tea.KeyMsg); ok {
  //       fmt.Printf("Input received key: %s\n", k.String())
	// }
	var cmd tea.Cmd
	
	// 1. Let the library handle all the complex keyboard logic (backspace, etc.)
	t.Model, cmd = t.Model.Update(msg)
	
	// 2. Sync the library's internal state to your local Content field
	t.Content = t.Model.Value()

	return t, cmd 
}

func (t *TextInput) Focus() {
	t.Model.Focus()
}

func (t *TextInput) Blur() {
	t.Model.Blur()
}

func (t TextInput) GetValue() string { return t.Content }
func (t TextInput) IsFocusable() bool { return true }
func (t TextInput) GetID() string     { return t.ID }
func (t TextInput) GetAction() string { return "" }
func (t TextInput) GetType() string { return "text-input" }
