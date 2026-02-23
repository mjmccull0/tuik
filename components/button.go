package components

import (
  "github.com/charmbracelet/lipgloss"
  tea "github.com/charmbracelet/bubbletea"
)

type Button struct {
  ID      string
  Text    string
  Action  string // The View ID or Shell Command to trigger
  focused bool
}

func (b Button) Render(ctx Context) string {
  // Standard style
  btnStyle := lipgloss.NewStyle().
    Padding(0, 3).
    MarginTop(1).
    Border(lipgloss.RoundedBorder())

  // If focused, give it a distinct color (Purple/5)
  if b.focused {
    btnStyle = btnStyle.
      BorderForeground(lipgloss.Color("5")).
      Foreground(lipgloss.Color("5")).
      Bold(true)
  }

  return btnStyle.Render(ctx.Resolve(b.Text))
}

func (b *Button) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
  // Buttons usually don't have internal state changes on keys, 
  // they just wait for 'enter' which is handled by main.go
  return b, nil
}

func (b *Button) Focus() { b.focused = true }
func (b *Button) Blur()  { b.focused = false }
func (b *Button) IsFocused() bool { return b.focused }
func (b Button) IsFocusable() bool { return true }
func (b Button) GetID() string     { return b.ID }
func (b Button) GetAction() string { return b.Action }
func (b Button) GetValue() string  { return "" }
func (b Button) GetType() string   { return "button" }
