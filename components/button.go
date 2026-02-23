package components

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type Button struct {
	ID      string
	Text    string
	Action  string 
	focused bool
	Styles  StyleConfig
}

func (b *Button) Render(ctx Context) string {
	text := ctx.Resolve(b.Text)
	
	// Start with the styles defined in JSON (margins, width, etc.)
	style := b.Styles.ToLipgloss().Padding(0, 2)

	if b.focused {
		// Apply Focus styling
		style = style.
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("205")).
			Foreground(lipgloss.Color("205")).
			Bold(true)
		
		return style.Render("󰄾 " + text)
	}

	// Apply Normal styling
	style = style.
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	return style.Render("  " + text)
}

func (b *Button) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
	if !b.focused {
		return b, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			return b, func() tea.Msg {
				return ActionMsg{ID: b.ID, Action: b.Action}
			}
		}
	}
	return b, nil
}

func (b *Button) Focus()           { b.focused = true }
func (b *Button) Blur()            { b.focused = false }
func (b *Button) IsFocused() bool  { return b.focused }
func (b *Button) IsFocusable() bool { return true }
func (b *Button) GetID() string     { return b.ID }
func (b *Button) GetAction() string { return b.Action }
func (b *Button) GetValue() string  { return "" }
func (b *Button) GetType() string   { return "button" }
