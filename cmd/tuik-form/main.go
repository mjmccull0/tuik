package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- INTERFACE DEFINITION ---
type Field interface {
	Update(tea.Msg) (Field, tea.Cmd)
	View() string
	Focus() tea.Cmd
	Blur()
	Value() string
}

// --- COMPONENT 1: TEXT INPUT ---
type inputField struct{ model textinput.Model }

func (f *inputField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	f.model, cmd = f.model.Update(msg)
	return f, cmd
}
func (f *inputField) View() string  { return f.model.View() }
func (f *inputField) Focus() tea.Cmd { return f.model.Focus() }
func (f *inputField) Blur()         { f.model.Blur() }
func (f *inputField) Value() string { return f.model.Value() }

// --- COMPONENT 2: TEXT AREA ---
type areaField struct{ model textarea.Model }

func (f *areaField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	f.model, cmd = f.model.Update(msg)
	return f, cmd
}
func (f *areaField) View() string  { return f.model.View() }
func (f *areaField) Focus() tea.Cmd { return f.model.Focus() }
func (f *areaField) Blur()         { f.model.Blur() }
func (f *areaField) Value() string { return f.model.Value() }

// --- COMPONENT 3: CHOICE LIST ---
type choiceField struct {
	options []string
	index   int
	focused bool
}

func (f *choiceField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "up", "k":
			if f.index > 0 { f.index-- }
		case "down", "j":
			if f.index < len(f.options)-1 { f.index++ }
		}
	}
	return f, nil
}
func (f *choiceField) View() string {
	var s strings.Builder
	for i, opt := range f.options {
		if i == f.index {
			s.WriteString(fmt.Sprintf("> %s\n", opt))
		} else {
			s.WriteString(fmt.Sprintf("  %s\n", opt))
		}
	}
	return s.String()
}
func (f *choiceField) Focus() tea.Cmd { f.focused = true; return nil }
func (f *choiceField) Blur()          { f.focused = false }
func (f *choiceField) Value() string  { return f.options[f.index] }

// --- THE FORM CONTROLLER ---
type formModel struct {
	fields     []Field
	labels     []string
	focusIndex int
	submitted  bool
}

func (m formModel) Init() tea.Cmd {
	return m.fields[0].Focus()
}

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab", "shift+tab":
			m.fields[m.focusIndex].Blur()
			if msg.String() == "tab" {
				m.focusIndex = (m.focusIndex + 1) % len(m.fields)
			} else {
				m.focusIndex = (m.focusIndex - 1 + len(m.fields)) % len(m.fields)
			}
			return m, m.fields[m.focusIndex].Focus()
		case "enter":
			if m.focusIndex == len(m.fields)-1 {
				m.submitted = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	// Use the result of Update to update the slice
	m.fields[m.focusIndex], cmd = m.fields[m.focusIndex].Update(msg)
	return m, cmd
}

func (m formModel) View() string {
	if m.submitted {
		var res strings.Builder
		res.WriteString("Results:\n\n")
		for i, f := range m.fields {
			res.WriteString(fmt.Sprintf("%s: %s\n", m.labels[i], f.Value()))
		}
		return res.String()
	}

	var s strings.Builder
	for i, f := range m.fields {
		labelStyle := lipgloss.NewStyle().Bold(true)
		fieldStyle := lipgloss.NewStyle().MarginLeft(2)

		if i == m.focusIndex {
			labelStyle = labelStyle.Foreground(lipgloss.Color("205"))
		} else {
			labelStyle = labelStyle.Foreground(lipgloss.Color("240"))
		}

		s.WriteString(labelStyle.Render(m.labels[i]) + "\n")
		s.WriteString(fieldStyle.Render(f.View()) + "\n\n")
	}
	
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	s.WriteString(helpStyle.Render("(tab to move • j/k for list • enter on last field to submit)"))
	
	return lipgloss.NewStyle().Padding(1, 2).Render(s.String())
}

func main() {
	ti := textinput.New()
	ti.Placeholder = "Enter name..."
	
	ta := textarea.New()
	ta.Placeholder = "Enter bio..."

	// IMPORTANT: Pass pointers to the structs so they satisfy the interface
	m := formModel{
		labels: []string{"User Name", "Biography", "User Role"},
		fields: []Field{
			&inputField{ti},
			&areaField{ta},
			&choiceField{options: []string{"Admin", "Editor", "Viewer"}},
		},
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
