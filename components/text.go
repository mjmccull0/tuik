package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Text struct {
	Content string
	Color   string // Inline style
	Bold    bool   // Inline style
	Style   StyleConfig
}

func (t Text) Render(ctx Context) string {
	content := ctx.Resolve(t.Content)
	return content
}

func (t Text) Blur() {}
func (t Text) Focus() {}
func (t Text) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) { return t, nil }
func (t Text) IsFocusable() bool { return false }
func (t Text) GetType() string   { return "text" }
func (t Text) GetID() string     { return "" }
func (t Text) GetValue() string  { return "" }
func (t Text) GetAction() string { return "" }
