package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- FIELD INTERFACE ---
type Field interface {
	Update(tea.Msg) (Field, tea.Cmd)
	View() string
	Focus() tea.Cmd
	Blur()
	Value() string
}

// --- COMPONENTS ---
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

type choiceField struct {
	options []string
	index   int
	focused bool
}
func (f *choiceField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused { return f, nil }
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "up", "k": if f.index > 0 { f.index-- }
		case "down", "j": if f.index < len(f.options)-1 { f.index++ }
		}
	}
	return f, nil
}
func (f *choiceField) View() string {
	var s strings.Builder
	for i, opt := range f.options {
		if i == f.index { s.WriteString(fmt.Sprintf("> %s\n", opt)) } else { s.WriteString(fmt.Sprintf("  %s\n", opt)) }
	}
	return s.String()
}
func (f *choiceField) Focus() tea.Cmd { f.focused = true; return nil }
func (f *choiceField) Blur()          { f.focused = false }
func (f *choiceField) Value() string  { return f.options[f.index] }

// --- FORM CONTROLLER ---
type formModel struct {
	fields     []Field
	labels     []string
	keys       []string
	focusIndex int
	submitted  bool
}

func (m formModel) Init() tea.Cmd { return m.fields[0].Focus() }

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c": return m, tea.Quit
		case "tab", "shift+tab":
			m.fields[m.focusIndex].Blur()
			if msg.String() == "tab" { m.focusIndex = (m.focusIndex + 1) % len(m.fields) } else { m.focusIndex = (m.focusIndex - 1 + len(m.fields)) % len(m.fields) }
			return m, m.fields[m.focusIndex].Focus()
		case "enter":
			if m.focusIndex == len(m.fields)-1 { m.submitted = true; return m, tea.Quit }
		}
	}
	var cmd tea.Cmd
	m.fields[m.focusIndex], cmd = m.fields[m.focusIndex].Update(msg)
	return m, cmd
}

func (m formModel) View() string {
	var s strings.Builder
	for i, f := range m.fields {
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
		if i == m.focusIndex { labelStyle = labelStyle.Foreground(lipgloss.Color("205")) }
		s.WriteString(labelStyle.Render(m.labels[i]) + "\n")
		s.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(f.View()) + "\n\n")
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(s.String())
}

// --- CLI HANDLING ---
type fieldArgs []string
func (i *fieldArgs) String() string { return "fields" }
func (i *fieldArgs) Set(value string) error { *i = append(*i, value); return nil }

func main() {
	var fArgs fieldArgs
	flag.Var(&fArgs, "f", "field definition 'key:type:label[:options]'")
	flag.Parse()

	if len(fArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: tuik-form -f 'key:type:label[:options]'")
		os.Exit(1)
	}

	m := formModel{}
	for _, arg := range fArgs {
		p := strings.Split(arg, ":")
		if len(p) < 3 { continue }
		key, fType, label := p[0], p[1], p[2]
		m.keys = append(m.keys, key)
		m.labels = append(m.labels, label)

		switch fType {
		case "input":
			ti := textinput.New(); ti.Placeholder = label
			m.fields = append(m.fields, &inputField{ti})
		case "area":
			ta := textarea.New(); ta.Placeholder = label
			m.fields = append(m.fields, &areaField{ta})
		case "choice":
			opts := strings.Split(p[3], ",")
			m.fields = append(m.fields, &choiceField{options: opts})
		}
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, _ := p.Run()

	if fm, ok := finalModel.(formModel); ok && fm.submitted {
		out := make(map[string]string)
		for i, f := range fm.fields { out[fm.keys[i]] = f.Value() }
		json.NewEncoder(os.Stdout).Encode(out)
	}
}
