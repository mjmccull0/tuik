package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	Content     string
	IsFocusable bool
}

func (s *StaticComponent) Init() tea.Cmd                           { return nil }
func (s *StaticComponent) Update(msg tea.Msg) (Component, tea.Cmd) { return s, nil }
func (s *StaticComponent) View() string                            { return s.Content }
func (s *StaticComponent) SetSize(w, h int)                        {}
func (s *StaticComponent) Value() string                           { return "" }
func (s *StaticComponent) SetValue(v string)                       { s.Content = v }
func (s *StaticComponent) Focusable() bool                         { return s.IsFocusable }
func (s *StaticComponent) Focus() tea.Cmd                          { return nil }
func (s *StaticComponent) Blur()                                   {}

type Action struct {
	Pattern    string `yaml:"pattern"`
	ActionType string `yaml:"action_type"` // "set_state", "exec", "push_screen", "back"
	StateKey   string `yaml:"key"`         // For set_state
	Value      string `yaml:"value"`       // For set_state
	Target     string `yaml:"target"`      // For exec or push_screen (Screen ID)
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

type DropdownComponent struct {
	Options   []string
	Cursor    int
	IsFocused bool
	Width, Height int
}

func (d *DropdownComponent) Init() tea.Cmd      { return nil }
func (d *DropdownComponent) Focusable() bool    { return true }
func (d *DropdownComponent) Focus() tea.Cmd     { d.IsFocused = true; return nil }
func (d *DropdownComponent) Blur()              { d.IsFocused = false }
func (d *DropdownComponent) Value() string {
	if len(d.Options) > 0 && d.Cursor < len(d.Options) {
		return d.Options[d.Cursor]
	}
	return ""
}
func (d *DropdownComponent) SetValue(v string) {
	d.Options = strings.Split(strings.TrimSpace(v), "\n")
	if d.Cursor >= len(d.Options) {
		d.Cursor = 0
	}
}
func (d *DropdownComponent) SetSize(w, h int) { d.Width = w; d.Height = h }
func (d *DropdownComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !d.IsFocused {
		return d, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.Cursor > 0 { d.Cursor-- }
		case "down", "j":
			if d.Cursor < len(d.Options)-1 { d.Cursor++ }
		}
	}
	return d, nil
}
func (d *DropdownComponent) View() string {
	if len(d.Options) == 0 {
		return "[ No Options ]"
	}

	current := d.Options[d.Cursor]
	if !d.IsFocused {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("[ ") + current + lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(" ] \u25bc")
	}

	var s strings.Builder
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	for i, opt := range d.Options {
		if i == d.Cursor {
			// Clear pink chevron for the selected item
			s.WriteString(selectedStyle.Render("> " + opt) + "\n")
		} else {
			// Dimmed grey for other items
			s.WriteString(unselectedStyle.Render("  " + opt) + "\n")
		}
		if i >= d.Height-1 {
			break
		}
	}
	return s.String()
}

type ToggleComponent struct {
	Label     string
	On        bool
	IsFocused bool
}

func (t *ToggleComponent) Init() tea.Cmd      { return nil }
func (t *ToggleComponent) Focusable() bool    { return true }
func (t *ToggleComponent) Focus() tea.Cmd     { t.IsFocused = true; return nil }
func (t *ToggleComponent) Blur()              { t.IsFocused = false }
func (t *ToggleComponent) Value() string {
	if t.On { return "on" }
	return ""
}
func (t *ToggleComponent) SetValue(v string) { t.Label = v }
func (t *ToggleComponent) SetSize(w, h int)  {}
func (t *ToggleComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !t.IsFocused { return t, nil }
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == " " || msg.String() == "enter" {
			t.On = !t.On
		}
	}
	return t, nil
}
func (t *ToggleComponent) View() string {
	box := "[ ]"
	if t.On { box = "[X]" }
	style := lipgloss.NewStyle()
	if t.IsFocused { style = style.Foreground(lipgloss.Color("205")) }
	return style.Render(box + " " + t.Label)
}

type RadioComponent struct {
	Options   []string
	Cursor    int
	Selected  int
	IsFocused bool
	Width, Height int
}

func (r *RadioComponent) Init() tea.Cmd      { return nil }
func (r *RadioComponent) Focusable() bool    { return true }
func (r *RadioComponent) Focus() tea.Cmd     { r.IsFocused = true; return nil }
func (r *RadioComponent) Blur()              { r.IsFocused = false }
func (r *RadioComponent) Value() string {
	if len(r.Options) > 0 { return r.Options[r.Selected] }
	return ""
}
func (r *RadioComponent) SetValue(v string) {
	r.Options = strings.Split(strings.TrimSpace(v), "\n")
}
func (r *RadioComponent) SetSize(w, h int) { r.Width = w; r.Height = h }
func (r *RadioComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !r.IsFocused { return r, nil }
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if r.Cursor > 0 { r.Cursor-- }
		case "down", "j":
			if r.Cursor < len(r.Options)-1 { r.Cursor++ }
		case " ", "enter":
			r.Selected = r.Cursor
		}
	}
	return r, nil
}
func (r *RadioComponent) View() string {
	var s strings.Builder
	for i, opt := range r.Options {
		prefix := "( ) "
		if i == r.Selected { prefix = "(*) " }
		line := prefix + opt
		if i == r.Cursor && r.IsFocused {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+line) + "\n")
		} else {
			s.WriteString("  " + line + "\n")
		}
	}
	return s.String()
}

type MultiSelectComponent struct {
	Options   []string
	Cursor    int
	Selected  map[int]bool
	IsFocused bool
	Width, Height int
}

func (m *MultiSelectComponent) Init() tea.Cmd      { return nil }
func (m *MultiSelectComponent) Focusable() bool    { return true }
func (m *MultiSelectComponent) Focus() tea.Cmd     { m.IsFocused = true; return nil }
func (m *MultiSelectComponent) Blur()              { m.IsFocused = false }
func (m *MultiSelectComponent) Value() string {
	var res []string
	for i, opt := range m.Options {
		if m.Selected[i] { res = append(res, opt) }
	}
	return strings.Join(res, " ")
}
func (m *MultiSelectComponent) SetValue(v string) {
	m.Options = strings.Split(strings.TrimSpace(v), "\n")
	if m.Selected == nil { m.Selected = make(map[int]bool) }
}
func (m *MultiSelectComponent) SetSize(w, h int) { m.Width = w; m.Height = h }
func (m *MultiSelectComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !m.IsFocused { return m, nil }
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 { m.Cursor-- }
		case "down", "j":
			if m.Cursor < len(m.Options)-1 { m.Cursor++ }
		case " ", "enter":
			m.Selected[m.Cursor] = !m.Selected[m.Cursor]
		}
	}
	return m, nil
}
func (m *MultiSelectComponent) View() string {
	var s strings.Builder
	for i, opt := range m.Options {
		box := "[ ] "
		if m.Selected[i] { box = "[X] " }
		line := box + opt
		if i == m.Cursor && m.IsFocused {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+line) + "\n")
		} else {
			s.WriteString("  " + line + "\n")
		}
	}
	return s.String()
}

type textareaWrapper struct {
	model       textarea.Model
	IsSecure    bool
	History     []string
	HistoryIdx  int
	IsFocused   bool
}

func (t *textareaWrapper) Init() tea.Cmd { return textarea.Blink }
func (t *textareaWrapper) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !t.IsFocused {
		return t, nil
	}
	var cmd tea.Cmd
	
	// History Navigation: Ctrl+P / Ctrl+N to cycle history
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if len(t.History) > 0 {
			switch msg.String() {
			case "ctrl+p": // Previous History
				t.HistoryIdx = (t.HistoryIdx + 1) % len(t.History)
				t.model.SetValue(t.History[t.HistoryIdx])
				return t, nil
			case "ctrl+n": // Next History
				t.HistoryIdx = (t.HistoryIdx - 1 + len(t.History)) % len(t.History)
				t.model.SetValue(t.History[t.HistoryIdx])
				return t, nil
			}
		}
	}

	t.model, cmd = t.model.Update(msg)
	return t, cmd
}

func (t *textareaWrapper) View() string {
	if t.IsSecure {
		val := t.model.Value()
		masked := strings.Repeat("*", len(val))
		// We still want the cursor to show up
		return masked
	}
	return t.model.View()
}

func (t *textareaWrapper) SetSize(w, h int)       { t.model.SetWidth(w); t.model.SetHeight(h) }
func (t *textareaWrapper) Value() string          { return t.model.Value() }
func (t *textareaWrapper) SetValue(v string)      { t.model.SetValue(v) }
func (t *textareaWrapper) Focusable() bool        { return true }
func (t *textareaWrapper) Focus() tea.Cmd {
	t.IsFocused = true
	return t.model.Focus()
}
func (t *textareaWrapper) Blur() {
	t.IsFocused = false
	t.model.Blur()
}

// --- Configuration & Model ---

type Config struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	WidthPC     int      `yaml:"width_pc"`
	HeightPC    int      `yaml:"height_pc"`
	Direction   string   `yaml:"direction"`
	Panes       []Config `yaml:"panes"`
	Cmd         string   `yaml:"cmd"`
	Watches     string   `yaml:"watches"`
	Type        string   `yaml:"type"`
	Actions     []Action `yaml:"actions"`
	Focusable   *bool    `yaml:"focusable"`
	Secure      bool     `yaml:"secure"`
	HistoryFile string   `yaml:"history_file"`
	Shortcut    string   `yaml:"shortcut"`
	Src         string   `yaml:"src"`
}

type TopLevelConfig struct {
	Screens       []Config `yaml:"screens"`
	ShowShortcuts bool     `yaml:"show_shortcuts"`
}

type pane struct {
	model     Component
	conf      Config
	lastValue string
}

type screen struct {
	conf      Config
	flatPanes []*pane
	focusIdx  int
}

type ReloadMsg struct {
	Top TopLevelConfig
}

type model struct {
	configPath    string
	topConf       TopLevelConfig
	screens       []*screen
	activeIdx     int
	navStack      []int // Stack of screen indices
	width, height int
	state         map[string]string
}

func (m *model) resolve(template string) string {
	res := template
	s := m.screens[m.activeIdx]
	// 1. Inject Global State
	for k, v := range m.state {
		res = strings.ReplaceAll(res, "{{state."+k+"}}", v)
		res = strings.ReplaceAll(res, "[[state."+k+"]]", v)
	}
	// 2. Inject Pane Values FROM ACTIVE SCREEN
	for _, p := range s.flatPanes {
		val := strings.TrimSpace(p.model.Value())
		val = strings.TrimSuffix(val, "/")
		res = strings.ReplaceAll(res, "{{"+p.conf.ID+"}}", val)
		if val == "" {
			res = strings.ReplaceAll(res, "[["+p.conf.ID+"]]", "")
		} else {
			res = strings.ReplaceAll(res, "[["+p.conf.ID+"]]", val)
		}
	}
	return res
}

func (m *model) saveHistory() {
	s := m.screens[m.activeIdx]
	for _, p := range s.flatPanes {
		if p.conf.HistoryFile != "" {
			val := strings.TrimSpace(p.model.Value())
			if val == "" {
				continue
			}
			content, _ := os.ReadFile(p.conf.HistoryFile)
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 && lines[len(lines)-1] == val {
				continue
			}
			f, _ := os.OpenFile(p.conf.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			f.WriteString(val + "\n")
			f.Close()
		}
	}
}

func (m *model) executeAndSet(targetID, shellCmd string) {
	s := m.screens[m.activeIdx]
	var target *pane
	for _, p := range s.flatPanes {
		if p.conf.ID == targetID {
			target = p
			break
		}
	}
	if target == nil {
		return
	}

	finalCmd := m.resolve(shellCmd)
	if strings.Contains(finalCmd, "{{") {
		return
	}

	cmd := exec.Command("sh", "-c", finalCmd)
	if cwd, ok := m.state["cwd"]; ok {
		cmd.Dir = cwd
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		target.model.SetValue(fmt.Sprintf("Shell Error: %v\nCommand: %s\nOutput: %s", err, finalCmd, string(out)))
		return
	}
	m.saveHistory()
	target.model.SetValue(string(out))
}

func (m *model) runCmd(targetID string, triggerMsg tea.Msg) {
	s := m.screens[m.activeIdx]
	var target *pane
	for _, p := range s.flatPanes {
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
		for _, p := range s.flatPanes {
			if p.conf.ID == target.conf.Watches {
				trigger = p
				break
			}
		}
	}
	if trigger == nil {
		trigger = target
	}
	triggerVal := strings.TrimSpace(trigger.model.Value())

	if trigger != nil && len(trigger.conf.Actions) > 0 {
		keyMsg, isKey := triggerMsg.(tea.KeyMsg)
		for _, action := range trigger.conf.Actions {
			if action.TriggerKey != "" {
				if !isKey || keyMsg.String() != action.TriggerKey {
					continue
				}
			}
			matched, _ := regexp.MatchString(action.Pattern, triggerVal)
			if matched {
				switch action.ActionType {
				case "push_screen":
					for i, sc := range m.screens {
						if sc.conf.ID == action.Target {
							m.navStack = append(m.navStack, m.activeIdx)
							m.activeIdx = i
							return
						}
					}
				case "back":
					if len(m.navStack) > 0 {
						m.activeIdx = m.navStack[len(m.navStack)-1]
						m.navStack = m.navStack[:len(m.navStack)-1]
					}
					return
				case "set_state":
					val := m.resolve(action.Value)
					m.state[action.StateKey] = filepath.Clean(val)
					for _, other := range s.flatPanes {
						if other.conf.Watches == trigger.conf.ID && other.conf.ID != trigger.conf.ID {
							other.model.SetValue("")
						}
					}
				case "exec", "":
					actTarget := action.Target
					if actTarget == "" {
						actTarget = target.conf.ID
					}
					if action.Cmd != "" {
						m.executeAndSet(actTarget, action.Cmd)
					}
				}
				return
			}
		}
	}
	if target.conf.Cmd != "" {
		m.executeAndSet(target.conf.ID, target.conf.Cmd)
	}
}

// --- Bubble Tea Interface ---

func (m *model) buildScreens(top TopLevelConfig) {
	var screens []*screen
	for _, sc := range top.Screens {
		var flat []*pane
		var flatten func(*Config)
		flatten = func(c *Config) {
			// YAML INCLUDE
			if c.Src != "" {
				subFile, err := os.ReadFile(c.Src)
				if err == nil {
					var subConfig Config
					yaml.Unmarshal(subFile, &subConfig)
					if c.ID == "" { c.ID = subConfig.ID }
					if c.Title == "" { c.Title = subConfig.Title }
					if c.Type == "" { c.Type = subConfig.Type }
					if c.Direction == "" { c.Direction = subConfig.Direction }
					if len(c.Panes) == 0 { c.Panes = subConfig.Panes }
					if c.Cmd == "" { c.Cmd = subConfig.Cmd }
					if c.Watches == "" { c.Watches = subConfig.Watches }
					if len(c.Actions) == 0 { c.Actions = subConfig.Actions }
					if c.Focusable == nil { c.Focusable = subConfig.Focusable }
					if c.HistoryFile == "" { c.HistoryFile = subConfig.HistoryFile }
					c.Secure = subConfig.Secure || c.Secure
				}
			}

			if len(c.Panes) == 0 {
				var comp Component
				switch c.Type {
				case "static":
					isFocusable := false
					if c.Focusable != nil { isFocusable = *c.Focusable }
					comp = &StaticComponent{Content: c.Title, IsFocusable: isFocusable}
				case "list":
					comp = &ListComponent{}
				case "dropdown":
					comp = &DropdownComponent{}
				case "toggle":
					comp = &ToggleComponent{}
				case "radio":
					comp = &RadioComponent{}
				case "multi":
					comp = &MultiSelectComponent{}
				default:
					ta := textarea.New()
					ta.Placeholder = "ID: " + c.ID
					tw := &textareaWrapper{model: ta, IsSecure: c.Secure}
					if c.HistoryFile != "" {
						content, _ := os.ReadFile(c.HistoryFile)
						if len(content) > 0 { tw.History = strings.Split(strings.TrimSpace(string(content)), "\n") }
					}
					comp = tw
				}
				flat = append(flat, &pane{model: comp, conf: *c})
			}
			for i := range c.Panes { flatten(&c.Panes[i]) }
		}
		localSc := sc
		flatten(&localSc)
		screens = append(screens, &screen{conf: localSc, flatPanes: flat})
	}
	m.screens = screens
	m.topConf = top
	
	// Re-run initial commands for the active screen
	if len(m.screens) > m.activeIdx {
		s := m.screens[m.activeIdx]
		for _, p := range s.flatPanes {
			if p.conf.Cmd != "" && p.conf.Watches == "" {
				m.runCmd(p.conf.ID, nil)
			}
			if p.model.Focusable() && s.focusIdx == 0 {
				for i, p2 := range s.flatPanes {
					if p2.model.Focusable() { s.focusIdx = i; break }
				}
			}
		}
		s.flatPanes[s.focusIdx].model.Focus()
	}
}

func watchConfig(path string, p *tea.Program) {
	var lastMod time.Time
	for {
		time.Sleep(1 * time.Second)
		info, err := os.Stat(path)
		if err != nil { continue }
		if lastMod.IsZero() {
			lastMod = info.ModTime()
			continue
		}
		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime()
			file, _ := os.ReadFile(path)
			var top TopLevelConfig
			err := yaml.Unmarshal(file, &top)
			if err != nil {
				// Try backward compatibility
				var root Config
				if yaml.Unmarshal(file, &root) == nil {
					top.Screens = []Config{root}
				}
			}
			if len(top.Screens) > 0 {
				p.Send(ReloadMsg{Top: top})
			}
		}
	}
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sReload, ok := msg.(ReloadMsg); ok {
		m.buildScreens(sReload.Top)
		return m, nil
	}

	s := m.screens[m.activeIdx]
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if len(m.navStack) > 0 {
				m.activeIdx = m.navStack[len(m.navStack)-1]
				m.navStack = m.navStack[:len(m.navStack)-1]
				return m, nil
			}
			return m, tea.Quit
		case "tab":
			s.flatPanes[s.focusIdx].model.Blur()
			for {
				s.focusIdx = (s.focusIdx + 1) % len(s.flatPanes)
				if s.flatPanes[s.focusIdx].model.Focusable() {
					break
				}
			}
			return m, s.flatPanes[s.focusIdx].model.Focus()
		}

		// Check dynamic screen shortcuts
		keyStr := msg.String()
		for i, sc := range m.screens {
			if sc.conf.Shortcut != "" && sc.conf.Shortcut == keyStr {
				s.flatPanes[s.focusIdx].model.Blur()
				m.activeIdx = i
				newS := m.screens[m.activeIdx]
				// Ensure initial commands run for the new screen
				for _, p := range newS.flatPanes {
					if p.conf.Cmd != "" && p.conf.Watches == "" {
						m.runCmd(p.conf.ID, nil)
					}
				}
				return m, newS.flatPanes[newS.focusIdx].model.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	idx := s.focusIdx
	var cmd tea.Cmd
	s.flatPanes[idx].model, cmd = s.flatPanes[idx].model.Update(msg)
	cmds = append(cmds, cmd)

	if _, isKey := msg.(tea.KeyMsg); isKey {
		m.runCmd(s.flatPanes[idx].conf.ID, msg)
	}

	for i := 0; i < 3; i++ {
		changed := false
		for _, p := range s.flatPanes {
			currentVal := p.model.Value()
			if currentVal != p.lastValue {
				p.lastValue = currentVal
				changed = true
				for _, other := range s.flatPanes {
					if other.conf.Watches == p.conf.ID {
						m.runCmd(other.conf.ID, nil)
					}
				}
				m.runCmd(p.conf.ID, nil)
			}
		}
		if !changed { break }
	}

	return m, tea.Batch(cmds...)
}

func (m *model) renderRecursive(conf Config, w, h int) string {
	s := m.screens[m.activeIdx]
	if len(conf.Panes) == 0 {
		var target *pane
		for _, p := range s.flatPanes {
			if p.conf.ID == conf.ID {
				target = p
				break
			}
		}
		if target == nil { return "ERR: " + conf.ID }

		if !target.model.Focusable() {
			target.model.SetSize(w, h)
			return target.model.View()
		}

		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(w - 2).
			Height(h - 3)

		if s.flatPanes[s.focusIdx].conf.ID == conf.ID {
			style = style.Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("205"))
		}

		target.model.SetSize(w-4, h-4)
		titleText := conf.Title
		if conf.ID == "sidebar" {
			cwd := m.state["cwd"]
			if len(cwd) > 20 { cwd = "..." + cwd[len(cwd)-17:] }
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
	if m.width == 0 { return "Calculating..." }
	s := m.screens[m.activeIdx]

	var tabs []string
	for i, sc := range m.screens {
		title := sc.conf.Title
		if title == "" {
			title = sc.conf.ID
		}
		
		// Show shortcut only if enabled in config
		label := title
		if m.topConf.ShowShortcuts && sc.conf.Shortcut != "" {
			label = fmt.Sprintf("[%s] %s", sc.conf.Shortcut, title)
		}

		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.activeIdx {
			style = style.Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Bold(true)
		} else {
			style = style.Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255"))
		}
		tabs = append(tabs, style.Render(label))
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	return tabBar + "\n" + m.renderRecursive(s.conf, m.width, m.height-2)
}

func main() {
	configPath := flag.String("config", "", "Path to YAML")
	flag.Parse()

	if *configPath == "" {
		fmt.Println("Usage: -config <path>")
		os.Exit(1)
	}

	file, _ := os.ReadFile(*configPath)
	var top TopLevelConfig
	err := yaml.Unmarshal(file, &top)
	
	// Backward compatibility: try parsing as single Config
	if err != nil || len(top.Screens) == 0 {
		var root Config
		yaml.Unmarshal(file, &root)
		top.Screens = []Config{root}
	}

	var screens []*screen
	for _, sc := range top.Screens {
		var flat []*pane
		var flatten func(*Config)
		flatten = func(c *Config) {
			// YAML INCLUDE: Load from external file if 'src' is present
			if c.Src != "" {
				subFile, err := os.ReadFile(c.Src)
				if err == nil {
					var subConfig Config
					yaml.Unmarshal(subFile, &subConfig)
					// Merge subConfig into current config
					if c.ID == "" { c.ID = subConfig.ID }
					if c.Title == "" { c.Title = subConfig.Title }
					if c.Type == "" { c.Type = subConfig.Type }
					if c.Direction == "" { c.Direction = subConfig.Direction }
					if len(c.Panes) == 0 { c.Panes = subConfig.Panes }
					if c.Cmd == "" { c.Cmd = subConfig.Cmd }
					if c.Watches == "" { c.Watches = subConfig.Watches }
					if len(c.Actions) == 0 { c.Actions = subConfig.Actions }
					if c.Focusable == nil { c.Focusable = subConfig.Focusable }
					if c.HistoryFile == "" { c.HistoryFile = subConfig.HistoryFile }
					c.Secure = subConfig.Secure || c.Secure
				}
			}

			if len(c.Panes) == 0 {
				var comp Component
				switch c.Type {
				case "static":
					isFocusable := false
					if c.Focusable != nil { isFocusable = *c.Focusable }
					comp = &StaticComponent{Content: c.Title, IsFocusable: isFocusable}
				case "list":
					comp = &ListComponent{}
				case "dropdown":
					comp = &DropdownComponent{}
				case "toggle":
					comp = &ToggleComponent{}
				case "radio":
					comp = &RadioComponent{}
				case "multi":
					comp = &MultiSelectComponent{}
				default:
					ta := textarea.New()
					ta.Placeholder = "ID: " + c.ID
					tw := &textareaWrapper{model: ta, IsSecure: c.Secure}
					if c.HistoryFile != "" {
						content, _ := os.ReadFile(c.HistoryFile)
						if len(content) > 0 { tw.History = strings.Split(strings.TrimSpace(string(content)), "\n") }
					}
					comp = tw
				}
				flat = append(flat, &pane{model: comp, conf: *c})
			}
			for i := range c.Panes { flatten(&c.Panes[i]) }
		}
		// Since we need to modify the config during flattening (merging 'src'), 
		// we copy the screen config to a local variable we can take a pointer to.
		localSc := sc
		flatten(&localSc)
		screens = append(screens, &screen{conf: localSc, flatPanes: flat})
	}

	m := &model{
		configPath: *configPath,
		topConf:    top,
		state:      make(map[string]string),
	}
	m.state["cwd"], _ = os.Getwd()
	m.buildScreens(top)

	p := tea.NewProgram(m, tea.WithAltScreen())
	go watchConfig(*configPath, p)

	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
