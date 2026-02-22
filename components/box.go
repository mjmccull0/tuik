package components

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type Box struct {
	Styles StyleConfig
	Child  Component
}

func (b Box) Render(ctx RenderContext) string {
	content := b.Child.Render(ctx)
	style := lipgloss.NewStyle()

	if b.Styles.Padding > 0 { style = style.Padding(b.Styles.Padding) }
	if b.Styles.Margin > 0 { style = style.Margin(b.Styles.Margin) }
	if b.Styles.BackgroundColor != "" { style = style.Background(lipgloss.Color(b.Styles.BackgroundColor)) }
	if b.Styles.Border {
		style = style.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color(b.Styles.BorderColor))
	}
	return style.Render(content)
}

func (b Box) Update(msg tea.Msg) (Component, tea.Cmd) {
	newChild, cmd := b.Child.Update(msg)
	return Box{Styles: b.Styles, Child: newChild}, cmd
}

func (b Box) IsFocusable() bool { return b.Child.IsFocusable() }
func (b Box) GetType() string   { return b.Child.GetType() }
func (b Box) GetID() string     { return b.Child.GetID() }
func (b Box) GetValue() string  { return b.Child.GetValue() }
func (b Box) GetAction() string { return b.Child.GetAction() }
