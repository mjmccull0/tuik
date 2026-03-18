package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
)

// --- Component System ---

type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
	SetSize(w, h int)
	Value() string
	SetValue(string) // Needed for runCmd to update content
	Focusable() bool
	Focus() tea.Cmd
	Blur()
}

// StaticComponent: For headers/ASCII art
type StaticComponent struct {
	Content string
}

func (s *StaticComponent) Init() tea.Cmd                           { return nil }
func (s *StaticComponent) Update(msg tea.Msg) (Component, tea.Cmd) { return s, nil }
func (s *StaticComponent) View() string                            { return s.Content }
func (s *StaticComponent) SetSize(w, h int)                        {}
func (s *StaticComponent) Value() string                           { return "" }
func (s *StaticComponent) SetValue(v string)                       { s.Content = v }
func (s *StaticComponent) Focusable() bool                         { return false }
func (s *StaticComponent) Focus() tea.Cmd                          { return nil }
func (s *StaticComponent) Blur()                                   {}

// textareaWrapper: Adapts the Bubbles textarea to our Component interface
type textareaWrapper struct {
	model textarea.Model
}

func (t *textareaWrapper) Init() tea.Cmd { return textarea.Blink }
func (t *textareaWrapper) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}
func (t *textareaWrapper) View() string             { return t.model.View() }
func (t *textareaWrapper) SetSize(w, h int)         { t.model.SetWidth(w); t.model.SetHeight(h) }
func (t *textareaWrapper) Value() string            { return t.model.Value() }
func (t *textareaWrapper) SetValue(v string)        { t.model.SetValue(v) }
func (t *textareaWrapper) Focusable() bool          { return true }
func (t *textareaWrapper) Focus() tea.Cmd           { return t.model.Focus() }
func (t *textareaWrapper) Blur()                    { t.model.Blur() }

// --- Configuration ---

type Config struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	WidthPC   int      `yaml:"width_pc"`
	HeightPC  int      `yaml:"height_pc"`
	Direction string   `yaml:"direction"`
	Panes     []Config `yaml:"panes"`
	Cmd       string   `yaml:"cmd"`
	Watches   string   `yaml:"watches"`
	Type      string   `yaml:"type"`
}

type pane struct {
	model     Component
	conf      Config
	lastValue string
}

type model struct {
	rootConf   Config
	flatPanes  []*pane
	focusIndex int
	width      int
	height     int
}

// --- Logic ---

func (m *model) runCmd(targetID string) {
	var target *pane
	var trigger *pane

	for _, p := range m.flatPanes {
		if p.conf.ID == targetID { target = p }
	}
	if target == nil || target.conf.Watches == "" || target.conf.Cmd == "" { return }

	for _, p := range m.flatPanes {
		if p.conf.ID == target.conf.Watches { trigger = p }
	}
	if trigger == nil { return }

	triggerVal := strings.TrimSpace(trigger.model.Value())
	shellCmd := strings.ReplaceAll(target.conf.Cmd, "{{"+trigger.conf.ID+"}}", triggerVal)

	out, _ := exec.Command("sh", "-c", shellCmd).CombinedOutput()
	target.model.SetValue(string(out))
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab":
			m.flatPanes[m.focusIndex].model.Blur()
			// Find next focusable
			for {
				m.focusIndex = (m.focusIndex + 1) % len(m.flatPanes)
				if m.flatPanes[m.focusIndex].model.Focusable() {
					break
				}
			}
			return m, m.flatPanes[m.focusIndex].model.Focus()
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	idx := m.focusIndex
	var cmd tea.Cmd
	m.flatPanes[idx].model, cmd = m.flatPanes[idx].model.Update(msg)
	cmds = append(cmds, cmd)

	currentVal := m.flatPanes[idx].model.Value()
	if currentVal != m.flatPanes[idx].lastValue {
		m.flatPanes[idx].lastValue = currentVal
		for _, p := range m.flatPanes {
			if p.conf.Watches == m.flatPanes[idx].conf.ID {
				m.runCmd(p.conf.ID)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *model) renderRecursive(conf Config, w, h int) string {
	if len(conf.Panes) == 0 {
		var target *pane
		for _, p := range m.flatPanes {
			if p.conf.ID == conf.ID { target = p; break }
		}

		// If it's a static/design element, render it without border and title
		if !target.model.Focusable() {
			target.model.SetSize(w, h)
			return target.model.View()
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(w - 2).
			Height(h - 3)

		if m.flatPanes[m.focusIndex].conf.ID == conf.ID {
			style = style.Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("205"))
		}

		// Use the interface method SetSize
		target.model.SetSize(w-4, h-4)

		title := lipgloss.NewStyle().Bold(true).Render(" " + conf.Title + " ")
		return title + "\n" + style.Render(target.model.View())
	}

	var children []string
	for _, child := range conf.Panes {
		childW, childH := w, h
		if conf.Direction == "horizontal" {
			childW = (w * child.WidthPC) / 100
		} else {
			childH = (h * child.HeightPC) / 100
		}
		children = append(children, m.renderRecursive(child, childW, childH))
	}

	if conf.Direction == "horizontal" {
		return lipgloss.JoinHorizontal(lipgloss.Top, children...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, children...)
}

func (m *model) View() string {
	if m.width == 0 { return "Calculating..." }
	return m.renderRecursive(m.rootConf, m.width, m.height)
}

func main() {
	configPath := flag.String("config", "", "Path to YAML")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("Usage: -config <path>")
		os.Exit(1)
	}

	file, _ := os.ReadFile(*configPath)
	var root Config
	yaml.Unmarshal(file, &root)

	var flat []*pane
	var flatten func(Config)
	flatten = func(c Config) {
		if len(c.Panes) == 0 {
			var comp Component
			if c.Type == "static" {
				comp = &StaticComponent{Content: c.Title}
			} else {
				ta := textarea.New()
				ta.Placeholder = "ID: " + c.ID
				comp = &textareaWrapper{model: ta}
			}

			p := &pane{model: comp, conf: c}
			flat = append(flat, p)
		}
		for _, child := range c.Panes {
			flatten(child)
		}
	}
	flatten(root)

	// Focus first focusable pane
	for i, p := range flat {
		if p.model.Focusable() {
			m := &model{rootConf: root, flatPanes: flat, focusIndex: i}
			tea.NewProgram(m, tea.WithAltScreen()).Run()
			return
		}
	}
}
