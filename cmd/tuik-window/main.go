package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	SetValue(string)
	Focusable() bool
	Focus() tea.Cmd
	Blur()
}

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

type Action struct {
	Pattern    string `yaml:"pattern"`
	ActionType string `yaml:"action_type"` // "set_state" or "exec" (default)
	StateKey   string `yaml:"key"`         // For set_state
	Value      string `yaml:"value"`       // For set_state
	Target     string `yaml:"target"`      // For exec
	Cmd        string `yaml:"cmd"`         // For exec
	TriggerKey string `yaml:"trigger_key"` // e.g. "enter"
}

type ListComponent struct {
	Items         []string
	Cursor        int
	Width, Height int
	IsFocused     bool
}

func (l *ListComponent) Init() tea.Cmd      { return nil }
func (l *ListComponent) Focusable() bool    { return true }
func (l *ListComponent) Focus() tea.Cmd     { l.IsFocused = true; return nil }
func (l *ListComponent) Blur()              { l.IsFocused = false }
func (l *ListComponent) Value() string {
	if len(l.Items) > 0 && l.Cursor < len(l.Items) {
		return l.Items[l.Cursor]
	}
	return ""
}
func (l *ListComponent) SetValue(v string) {
	l.Items = strings.Split(strings.TrimSpace(v), "\n")
	if l.Cursor >= len(l.Items) {
		l.Cursor = 0
	}
}
func (l *ListComponent) SetSize(w, h int) { l.Width = w; l.Height = h }
func (l *ListComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !l.IsFocused {
		return l, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if l.Cursor > 0 {
				l.Cursor--
			}
		case "down", "j":
			if l.Cursor < len(l.Items)-1 {
				l.Cursor++
			}
		}
	}
	return l, nil
}
func (l *ListComponent) View() string {
	if l.Height <= 0 {
		return ""
	}
	var s strings.Builder
	
	// Basic Scrolling: Calculate start/end indices based on cursor
	start := 0
	if l.Cursor >= l.Height-2 {
		start = l.Cursor - (l.Height - 3)
	}
	
	for i := start; i < len(l.Items); i++ {
		item := l.Items[i]
		if i == l.Cursor {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+item) + "\n")
		} else {
			s.WriteString("  " + item + "\n")
		}
		if i-start >= l.Height-2 {
			break
		}
	}
	return s.String()
}

type textareaWrapper struct {
	model textarea.Model
}

func (t *textareaWrapper) Init() tea.Cmd { return textarea.Blink }
func (t *textareaWrapper) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	t.model, cmd = t.model.Update(msg)
	return t, cmd
}
func (t *textareaWrapper) View() string           { return t.model.View() }
func (t *textareaWrapper) SetSize(w, h int)       { t.model.SetWidth(w); t.model.SetHeight(h) }
func (t *textareaWrapper) Value() string          { return t.model.Value() }
func (t *textareaWrapper) SetValue(v string)      { t.model.SetValue(v) }
func (t *textareaWrapper) Focusable() bool        { return true }
func (t *textareaWrapper) Focus() tea.Cmd         { return t.model.Focus() }
func (t *textareaWrapper) Blur()                  { t.model.Blur() }

// --- Configuration & Model ---

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
	Actions   []Action `yaml:"actions"`
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
	state      map[string]string
}

func (m *model) executeAndSet(targetID, shellCmd, triggerVal string, triggerID string) {
	var target *pane
	for _, p := range m.flatPanes {
		if p.conf.ID == targetID {
			target = p
			break
		}
	}
	if target == nil {
		return
	}

	finalCmd := shellCmd

	// 1. Perform state-based replacement (e.g. {{state.cwd}})
	for k, v := range m.state {
		finalCmd = strings.ReplaceAll(finalCmd, "{{state."+k+"}}", v)
	}

	// 2. Perform trigger replacement (e.g. {{sidebar}})
	if triggerID != "" {
		// SAFETY GASKET: Terminal commands often fail with trailing slashes (e.g. cat folder/)
		// We trim the slash so 'cat folder/' becomes 'cat folder'
		cleanVal := strings.TrimSuffix(triggerVal, "/")
		finalCmd = strings.ReplaceAll(finalCmd, "{{"+triggerID+"}}", cleanVal)
	}

	// 3. FINAL SAFETY: DO NOT run if placeholders are still present (avoids the "cat {{file_list}}" error)
	if strings.Contains(finalCmd, "{{") {
		return
	}

	cmd := exec.Command("sh", "-c", finalCmd)
	if cwd, ok := m.state["cwd"]; ok {
		cmd.Dir = cwd
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Just show the error in the viewer so we know why it failed
		target.model.SetValue(fmt.Sprintf("Shell Error: %v\nCommand: %s\nOutput: %s", err, finalCmd, string(out)))
		return
	}
	target.model.SetValue(string(out))
}

func (m *model) runCmd(targetID string, triggerMsg tea.Msg) {
	var target *pane
	for _, p := range m.flatPanes {
		if p.conf.ID == targetID {
			target = p
			break
		}
	}
	if target == nil {
		return
	}

	var trigger *pane
	if target.conf.Watches != "" {
		for _, p := range m.flatPanes {
			if p.conf.ID == target.conf.Watches {
				trigger = p
				break
			}
		}
	}

	// SELF-TRIGGER: If no watch is defined, the target triggers itself.
	// This allows actions like navigation within a single pane.
	if trigger == nil {
		trigger = target
	}

	var triggerVal string
	var triggerID string
	if trigger != nil {
		triggerVal = strings.TrimSpace(trigger.model.Value())
		triggerID = trigger.conf.ID
	}

	// --- Action Processing ---
	// If the trigger has actions, we check if the trigger value matches any pattern.
	if trigger != nil && len(trigger.conf.Actions) > 0 {
		keyMsg, isKey := triggerMsg.(tea.KeyMsg)
		for _, action := range trigger.conf.Actions {
			// 1. Check if the trigger key matches (if defined)
			if action.TriggerKey != "" {
				if !isKey || keyMsg.String() != action.TriggerKey {
					continue
				}
			}

			matched, _ := regexp.MatchString(action.Pattern, triggerVal)
			if matched {
				// Handle state updates if specified
				if action.ActionType == "set_state" && action.StateKey != "" {
					val := action.Value
					// Replace placeholders in the value too
					for k, v := range m.state {
						val = strings.ReplaceAll(val, "{{state."+k+"}}", v)
					}
					val = strings.ReplaceAll(val, "{{"+triggerID+"}}", strings.TrimSuffix(triggerVal, "/"))
					
					// CLEAN THE PATH: Use filepath to resolve ../ and ./
					m.state[action.StateKey] = filepath.Clean(val)

					// CLEAR WATCHING PANES: When we navigate, it's polite to clear the viewer
					for _, other := range m.flatPanes {
						if other.conf.Watches == trigger.conf.ID && other.conf.ID != trigger.conf.ID {
							other.model.SetValue("")
						}
					}
				}

				// Execute the command for the specified target (defaulting to current target if empty)
				actTarget := action.Target
				if actTarget == "" {
					actTarget = target.conf.ID
				}
				if action.Cmd != "" {
					m.executeAndSet(actTarget, action.Cmd, triggerVal, triggerID)
				}
				// We stop at the first match to allow specific-to-general precedence
				return
			}
		}
	}

	// Fallback to simple execution if no actions matched or were defined
	if target.conf.Cmd != "" {
		m.executeAndSet(target.conf.ID, target.conf.Cmd, triggerVal, triggerID)
	}
}

// --- Bubble Tea Interface ---

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
			for {
				m.focusIndex = (m.focusIndex + 1) % len(m.flatPanes)
				if m.flatPanes[m.focusIndex].model.Focusable() { break }
			}
			return m, m.flatPanes[m.focusIndex].model.Focus()
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	// 1. Update focused pane
	idx := m.focusIndex
	var cmd tea.Cmd
	m.flatPanes[idx].model, cmd = m.flatPanes[idx].model.Update(msg)
	cmds = append(cmds, cmd)

	// 2. IMMEDIATE KEY ACTION: If this was a key press, check for actions on the focused pane
	// This allows "enter" to work even if the cursor hasn't moved.
	if _, isKey := msg.(tea.KeyMsg); isKey {
		m.runCmd(m.flatPanes[idx].conf.ID, msg)
	}

	// 3. Cascade changes through watches
	for i := 0; i < 3; i++ {
		changed := false
		for _, p := range m.flatPanes {
			currentVal := p.model.Value()
			if currentVal != p.lastValue {
				p.lastValue = currentVal
				changed = true
				
				// Trigger everything that watches this pane
				for _, other := range m.flatPanes {
					if other.conf.Watches == p.conf.ID {
						m.runCmd(other.conf.ID, nil)
					}
				}
				
				// ALSO: check if this pane itself has actions that should trigger when it changes
				m.runCmd(p.conf.ID, nil)
			}
		}
		if !changed {
			break
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) renderRecursive(conf Config, w, h int) string {
	if len(conf.Panes) == 0 {
		var target *pane
		for _, p := range m.flatPanes {
			if p.conf.ID == conf.ID {
				target = p
				break
			}
		}

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

		target.model.SetSize(w-4, h-4)
		titleText := conf.Title
		if conf.ID == "sidebar" {
			cwd := m.state["cwd"]
			if len(cwd) > 20 {
				cwd = "..." + cwd[len(cwd)-17:]
			}
			// Show item count for the list
			if l, ok := target.model.(*ListComponent); ok {
				titleText = fmt.Sprintf(" %s [%d] (%s) ", conf.Title, len(l.Items), cwd)
			}
		}
		title := lipgloss.NewStyle().Bold(true).Render(titleText)
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
	if m.width == 0 {
		return "Calculating..."
	}
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
			switch c.Type {
			case "static":
				comp = &StaticComponent{Content: c.Title}
			case "list":
				comp = &ListComponent{}
			default:
				ta := textarea.New()
				ta.Placeholder = "ID: " + c.ID
				comp = &textareaWrapper{model: ta}
			}
			flat = append(flat, &pane{model: comp, conf: c})
		}
		for _, child := range c.Panes {
			flatten(child)
		}
	}
	flatten(root)

	m := &model{
		rootConf:  root,
		flatPanes: flat,
		state:     make(map[string]string),
	}

	m.state["cwd"], _ = os.Getwd()

	for _, p := range m.flatPanes {
		if p.conf.Cmd != "" && p.conf.Watches == "" {
			m.runCmd(p.conf.ID, nil)
		}
	}

	for i, p := range m.flatPanes {
		if p.model.Focusable() {
			m.focusIndex = i
			p.model.Focus()
			break
		}
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
