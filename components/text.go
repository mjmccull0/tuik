package components

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type Text struct {
	Content string
	Color   string // Inline style
	Bold    bool   // Inline style
}

func (t Text) Render(ctx RenderContext) string {
	style := lipgloss.NewStyle()
	if t.Color != "" { style = style.Foreground(lipgloss.Color(t.Color)) }
	if t.Bold { style = style.Bold(true) }
	
	return style.Render(t.Content)
}

func (t Text) Blur() {}
func (t Text) Focus() {}
func (t Text) Update(msg tea.Msg) (Component, tea.Cmd) { return t, nil }
func (t Text) IsFocusable() bool { return false }
func (t Text) GetType() string   { return "text" }
func (t Text) GetID() string     { return "" }
func (t Text) GetValue() string  { return "" }
func (t Text) GetAction() string { return "" }
