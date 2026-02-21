package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func RenderInput(value string, focused bool) string {
	ti := textinput.New()
	ti.SetValue(value)
	
	if focused {
		ti.Focus()
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Render(ti.View())
	}
	
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Render(ti.View())
}
