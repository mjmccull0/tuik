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
	ActionType string `yaml:"action_type"`
	StateKey   string `yaml:"key"`
	Value      string `yaml:"value"`
	Target     string `yaml:"target"`
	Cmd        string `yaml:"cmd"`
	TriggerKey string `yaml:"trigger_key"`
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
	if len(l.Items) > 0 && l.Cursor < len(l.Items) { return l.Items[l.Cursor] }
	return ""
}
func (l *ListComponent) SetValue(v string) {
	l.Items = strings.Split(strings.TrimSpace(v), "\n")
	if l.Cursor >= len(l.Items) { l.Cursor = 0 }
}
func (l *ListComponent) SetSize(w, h int) { l.Width = w; l.Height = h }
func (l *ListComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !l.IsFocused { return l, nil }
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k": if l.Cursor > 0 { l.Cursor-- }
		case "down", "j": if l.Cursor < len(l.Items)-1 { l.Cursor++ }
		}
	}
	return l, nil
}
func (l *ListComponent) View() string {
	if l.Height <= 0 { return "" }
	var s strings.Builder
	start := 0
	if l.Cursor >= l.Height-2 { start = l.Cursor - (l.Height - 3) }
	for i := start; i < len(l.Items); i++ {
		item := l.Items[i]
		if i == l.Cursor { s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+item) + "\n") } else { s.WriteString("  " + item + "\n") }
		if i-start >= l.Height-2 { break }
	}
	return strings.TrimSuffix(s.String(), "\n")
}

type DropdownComponent struct {
	Options       []string
	Cursor        int
	IsFocused     bool
	Width, Height int
}

func (d *DropdownComponent) Init() tea.Cmd      { return nil }
func (d *DropdownComponent) Focusable() bool    { return true }
func (d *DropdownComponent) Focus() tea.Cmd     { d.IsFocused = true; return nil }
func (d *DropdownComponent) Blur()              { d.IsFocused = false }
func (d *DropdownComponent) Value() string {
	if len(d.Options) > 0 && d.Cursor < len(d.Options) { return d.Options[d.Cursor] }
	return ""
}
func (d *DropdownComponent) SetValue(v string) {
	d.Options = strings.Split(strings.TrimSpace(v), "\n")
	if d.Cursor >= len(d.Options) { d.Cursor = 0 }
}
func (d *DropdownComponent) SetSize(w, h int) { d.Width = w; d.Height = h }
func (d *DropdownComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !d.IsFocused { return d, nil }
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k": if d.Cursor > 0 { d.Cursor-- }
		case "down", "j": if d.Cursor < len(d.Options)-1 { d.Cursor++ }
		}
	}
	return d, nil
}
func (d *DropdownComponent) View() string {
	if len(d.Options) == 0 { return "[ No Options ]" }
	current := d.Options[d.Cursor]
	if !d.IsFocused {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("[ ") + current + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" ] \u25bc")
	}
	var s strings.Builder
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	for i, opt := range d.Options {
		if i == d.Cursor { s.WriteString(selectedStyle.Render("> " + opt) + "\n") } else { s.WriteString(unselectedStyle.Render("  " + opt) + "\n") }
		if i >= d.Height-1 { break }
	}
	return strings.TrimSuffix(s.String(), "\n")
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
	if m, ok := msg.(tea.KeyMsg); ok && (m.String() == " " || m.String() == "enter") { t.On = !t.On }
	return t, nil
}
func (t *ToggleComponent) View() string {
	box := "[ ]"; if t.On { box = "[X]" }; style := lipgloss.NewStyle()
	if t.IsFocused { style = style.Foreground(lipgloss.Color("205")) }
	return style.Render(box + " " + t.Label)
}

type RadioComponent struct {
	Options          []string
	Cursor, Selected int
	IsFocused        bool
	Width, Height    int
}

func (r *RadioComponent) Init() tea.Cmd      { return nil }
func (r *RadioComponent) Focusable() bool    { return true }
func (r *RadioComponent) Focus() tea.Cmd     { r.IsFocused = true; return nil }
func (r *RadioComponent) Blur()              { r.IsFocused = false }
func (r *RadioComponent) Value() string {
	if len(r.Options) > 0 { return r.Options[r.Selected] }
	return ""
}
func (r *RadioComponent) SetValue(v string) { r.Options = strings.Split(strings.TrimSpace(v), "\n") }
func (r *RadioComponent) SetSize(w, h int)  { r.Width = w; r.Height = h }
func (r *RadioComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !r.IsFocused { return r, nil }
	if m, ok := msg.(tea.KeyMsg); ok {
		switch m.String() {
		case "up", "k": if r.Cursor > 0 { r.Cursor-- }
		case "down", "j": if r.Cursor < len(r.Options)-1 { r.Cursor++ }
		case " ", "enter": r.Selected = r.Cursor
		}
	}
	return r, nil
}
func (r *RadioComponent) View() string {
	var s strings.Builder
	for i, opt := range r.Options {
		prefix := "( ) "; if i == r.Selected { prefix = "(*) " }
		line := prefix + opt
		if i == r.Cursor && r.IsFocused { s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+line) + "\n") } else { s.WriteString("  " + line + "\n") }
	}
	return strings.TrimSuffix(s.String(), "\n")
}

type MultiSelectComponent struct {
	Options       []string
	Cursor        int
	Selected      map[int]bool
	IsFocused     bool
	Width, Height int
}

func (m *MultiSelectComponent) Init() tea.Cmd      { return nil }
func (m *MultiSelectComponent) Focusable() bool    { return true }
func (m *MultiSelectComponent) Focus() tea.Cmd     { m.IsFocused = true; return nil }
func (m *MultiSelectComponent) Blur()              { m.IsFocused = false }
func (m *MultiSelectComponent) Value() string {
	var res []string
	for i, opt := range m.Options { if m.Selected[i] { res = append(res, opt) } }
	return strings.Join(res, " ")
}
func (m *MultiSelectComponent) SetValue(v string) {
	m.Options = strings.Split(strings.TrimSpace(v), "\n")
	if m.Selected == nil { m.Selected = make(map[int]bool) }
}
func (m *MultiSelectComponent) SetSize(w, h int) { m.Width = w; m.Height = h }
func (m *MultiSelectComponent) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !m.IsFocused { return m, nil }
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k": if m.Cursor > 0 { m.Cursor-- }
		case "down", "j": if m.Cursor < len(m.Options)-1 { m.Cursor++ }
		case " ", "enter": m.Selected[m.Cursor] = !m.Selected[m.Cursor]
		}
	}
	return m, nil
}
func (m *MultiSelectComponent) View() string {
	var s strings.Builder
	for i, opt := range m.Options {
		box := "[ ] "; if m.Selected[i] { box = "[X] " }
		line := box + opt
		if i == m.Cursor && m.IsFocused { s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("> "+line) + "\n") } else { s.WriteString("  " + line + "\n") }
	}
	return strings.TrimSuffix(s.String(), "\n")
}

type textareaWrapper struct {
	model      textarea.Model
	IsSecure   bool
	History    []string
	HistoryIdx int
	IsFocused  bool
}

func (t *textareaWrapper) Init() tea.Cmd { return textarea.Blink }
func (t *textareaWrapper) Update(msg tea.Msg) (Component, tea.Cmd) {
	if !t.IsFocused { return t, nil }
	if key, ok := msg.(tea.KeyMsg); ok && len(t.History) > 0 {
		switch key.String() {
		case "ctrl+p": t.HistoryIdx = (t.HistoryIdx + 1) % len(t.History); t.model.SetValue(t.History[t.HistoryIdx]); return t, nil
		case "ctrl+n": t.HistoryIdx = (t.HistoryIdx - 1 + len(t.History)) % len(t.History); t.model.SetValue(t.History[t.HistoryIdx]); return t, nil
		}
	}
	var cmd tea.Cmd; t.model, cmd = t.model.Update(msg); return t, cmd
}
func (t *textareaWrapper) View() string {
	if t.IsSecure { return strings.Repeat("*", len(t.model.Value())) }
	return t.model.View()
}
func (t *textareaWrapper) SetSize(w, h int)       { t.model.SetWidth(w); t.model.SetHeight(h) }
func (t *textareaWrapper) Value() string          { return t.model.Value() }
func (t *textareaWrapper) SetValue(v string)      { t.model.SetValue(v) }
func (t *textareaWrapper) Focusable() bool        { return true }
func (t *textareaWrapper) Focus() tea.Cmd { t.IsFocused = true; return t.model.Focus() }
func (t *textareaWrapper) Blur() { t.IsFocused = false; t.model.Blur() }

// --- Configuration & Model ---

type Config struct {
	ID, Title, Direction, Cmd, Watches, Type, HistoryFile, Shortcut, Src string
	WidthPC, HeightPC                                                    int
	Items                                                                []Config
	Actions                                                              []Action
	Focusable                                                            *bool
	Secure                                                               bool
}

type TopLevelConfig struct {
	Views         []Config `yaml:"views"`
	ShowShortcuts bool     `yaml:"show_shortcuts"`
}

type node struct {
	model     Component
	conf      Config
	lastValue string
	x, y, w, h int
}

type view struct {
	conf     Config
	nodes    []*node
	focusIdx int
}

type ReloadMsg struct { Top TopLevelConfig }

type model struct {
	configPath    string
	topConf       TopLevelConfig
	views         []*view
	activeIdx     int
	navStack      []int
	width, height int
	state         map[string]string
	tabWidths     []int
}

func (m *model) resolve(template string) string {
	res := template; v := m.views[m.activeIdx]
	for k, val := range m.state { res = strings.ReplaceAll(res, "{{state."+k+"}}", val); res = strings.ReplaceAll(res, "[[state."+k+"]]", val) }
	for _, n := range v.nodes {
		val := strings.TrimSpace(n.model.Value()); val = strings.TrimSuffix(val, "/")
		res = strings.ReplaceAll(res, "{{"+n.conf.ID+"}}", val)
		if val == "" { res = strings.ReplaceAll(res, "[["+n.conf.ID+"]]", "") } else { res = strings.ReplaceAll(res, "[["+n.conf.ID+"]]", val) }
	}
	return res
}

func (m *model) executeAndSet(targetID, shellCmd string) {
	v := m.views[m.activeIdx]; var target *node
	for _, n := range v.nodes { if n.conf.ID == targetID { target = n; break } }
	if target == nil { return }
	finalCmd := m.resolve(shellCmd); if strings.Contains(finalCmd, "{{") { return }
	cmd := exec.Command("sh", "-c", finalCmd); if cwd, ok := m.state["cwd"]; ok { cmd.Dir = cwd }
	out, err := cmd.CombinedOutput()
	if err != nil {
		target.model.SetValue(fmt.Sprintf("Shell Error: %v\nCommand: %s\nOutput: %s", err, finalCmd, string(out)))
		return
	}
	for _, p := range v.nodes {
		if p.conf.HistoryFile != "" {
			val := strings.TrimSpace(p.model.Value()); if val == "" { continue }
			f, _ := os.OpenFile(p.conf.HistoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); f.WriteString(val + "\n"); f.Close()
		}
	}
	target.model.SetValue(string(out))
}

func (m *model) runCmd(targetID string, triggerMsg tea.Msg) {
	v := m.views[m.activeIdx]; var target *node
	for _, n := range v.nodes { if n.conf.ID == targetID { target = n; break } }
	if target == nil { return }
	var trigger *node
	if target.conf.Watches != "" {
		for _, n := range v.nodes { if n.conf.ID == target.conf.Watches { trigger = n; break } }
	}
	if trigger == nil { trigger = target }
	triggerVal := strings.TrimSpace(trigger.model.Value())
	if trigger != nil && len(trigger.conf.Actions) > 0 {
		keyMsg, isKey := triggerMsg.(tea.KeyMsg)
		for _, action := range trigger.conf.Actions {
			if action.TriggerKey != "" && (!isKey || keyMsg.String() != action.TriggerKey) { continue }
			matched, _ := regexp.MatchString(action.Pattern, triggerVal)
			if matched {
				switch action.ActionType {
				case "push_view":
					for i, vi := range m.views {
						if vi.conf.ID == action.Target { m.navStack = append(m.navStack, m.activeIdx); m.activeIdx = i; return }
					}
				case "back": if len(m.navStack) > 0 { m.activeIdx = m.navStack[len(m.navStack)-1]; m.navStack = m.navStack[:len(m.navStack)-1] }; return
				case "set_state":
					val := m.resolve(action.Value); m.state[action.StateKey] = filepath.Clean(val)
					for _, other := range v.nodes { if other.conf.Watches == trigger.conf.ID && other.conf.ID != trigger.conf.ID { other.model.SetValue("") } }
				case "exec", "":
					actTarget := action.Target; if actTarget == "" { actTarget = target.conf.ID }
					if action.Cmd != "" { m.executeAndSet(actTarget, action.Cmd) }
				}
				return
			}
		}
	}
	if target.conf.Cmd != "" { m.executeAndSet(target.conf.ID, target.conf.Cmd) }
}

func (m *model) buildViews(top TopLevelConfig) {
	var views []*view
	for _, vc := range top.Views {
		nodes := []*node{}
		var flatten func(*Config)
		flatten = func(c *Config) {
			if c.Src != "" {
				subFile, err := os.ReadFile(c.Src)
				if err == nil {
					var sc Config; yaml.Unmarshal(subFile, &sc)
					if c.ID == "" { c.ID = sc.ID }; if c.Title == "" { c.Title = sc.Title }; if c.Type == "" { c.Type = sc.Type }
					if c.Direction == "" { c.Direction = sc.Direction }; if len(c.Items) == 0 { c.Items = sc.Items }
					if c.Cmd == "" { c.Cmd = sc.Cmd }; if c.Watches == "" { c.Watches = sc.Watches }
					if len(c.Actions) == 0 { c.Actions = sc.Actions }; if c.Focusable == nil { c.Focusable = sc.Focusable }
					if c.HistoryFile == "" { c.HistoryFile = sc.HistoryFile }; c.Secure = sc.Secure || c.Secure
					// MERGE PERCENTAGES
					if c.WidthPC == 0 { c.WidthPC = sc.WidthPC }; if c.HeightPC == 0 { c.HeightPC = sc.HeightPC }
				}
			}
			if len(c.Items) == 0 {
				var comp Component; isFocus := false; if c.Focusable != nil { isFocus = *c.Focusable }
				switch c.Type {
				case "static": comp = &StaticComponent{Content: c.Title, IsFocusable: isFocus}
				case "list": comp = &ListComponent{}
				case "dropdown": comp = &DropdownComponent{}
				case "toggle": comp = &ToggleComponent{}
				case "radio": comp = &RadioComponent{}
				case "multi": comp = &MultiSelectComponent{}
				default:
					ta := textarea.New(); ta.Placeholder = "ID: " + c.ID; tw := &textareaWrapper{model: ta, IsSecure: c.Secure}
					if c.HistoryFile != "" { content, _ := os.ReadFile(c.HistoryFile); if len(content) > 0 { tw.History = strings.Split(strings.TrimSpace(string(content)), "\n") } }
					comp = tw
				}
				nodes = append(nodes, &node{model: comp, conf: *c})
			}
			for i := range c.Items { flatten(&c.Items[i]) }
		}
		localVc := vc; flatten(&localVc); views = append(views, &view{conf: localVc, nodes: nodes})
	}
	m.views = views; m.topConf = top
	if len(m.views) > m.activeIdx {
		v := m.views[m.activeIdx]
		for _, n := range v.nodes {
			if n.conf.Cmd != "" && n.conf.Watches == "" { m.runCmd(n.conf.ID, nil) }
			if n.model.Focusable() && v.focusIdx == 0 {
				for i, n2 := range v.nodes { if n2.model.Focusable() { v.focusIdx = i; break } }
			}
		}
		v.nodes[v.focusIdx].model.Focus()
	}
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if vReload, ok := msg.(ReloadMsg); ok { m.buildViews(vReload.Top); return m, nil }
	v := m.views[m.activeIdx]
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if msg.Y == 0 {
				curX := 0; for i, width := range m.tabWidths {
					if msg.X >= curX && msg.X < curX+width {
						v.nodes[v.focusIdx].model.Blur(); m.activeIdx = i
						return m, m.views[m.activeIdx].nodes[m.views[m.activeIdx].focusIdx].model.Focus()
					}
					curX += width
				}
			}
			for i, n := range v.nodes {
				if msg.X >= n.x && msg.X < n.x+n.w && msg.Y >= n.y && msg.Y < n.y+n.h {
					if d, ok := n.model.(*DropdownComponent); ok && d.IsFocused {
						startY := n.y + 1; if n.conf.Title != "" { startY++ }
						clickIdx := msg.Y - startY
						if clickIdx >= 0 && clickIdx < len(d.Options) { d.Cursor = clickIdx; return m, nil }
					}
					if n.model.Focusable() {
						v.nodes[v.focusIdx].model.Blur(); v.focusIdx = i; focusCmd := n.model.Focus()
						m.runCmd(n.conf.ID, tea.KeyMsg{Type: tea.KeyEnter}); return m, focusCmd
					}
				}
			}
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c": return m, tea.Quit
		case "esc": if len(m.navStack) > 0 { m.activeIdx = m.navStack[len(m.navStack)-1]; m.navStack = m.navStack[:len(m.navStack)-1]; return m, nil }
			return m, tea.Quit
		case "tab", "shift+tab":
			v.nodes[v.focusIdx].model.Blur(); isReverse := msg.String() == "shift+tab"
			for {
				if isReverse { v.focusIdx = (v.focusIdx - 1 + len(v.nodes)) % len(v.nodes) } else { v.focusIdx = (v.focusIdx + 1) % len(v.nodes) }
				if v.nodes[v.focusIdx].model.Focusable() { break }
			}
			return m, v.nodes[v.focusIdx].model.Focus()
		}
		keyStr := msg.String(); for i, vi := range m.views {
			if vi.conf.Shortcut != "" && vi.conf.Shortcut == keyStr {
				v.nodes[v.focusIdx].model.Blur(); m.activeIdx = i; newV := m.views[m.activeIdx]
				for _, n := range newV.nodes { if n.conf.Cmd != "" && n.conf.Watches == "" { m.runCmd(n.conf.ID, nil) } }
				return m, newV.nodes[newV.focusIdx].model.Focus()
			}
		}
	case tea.WindowSizeMsg: m.width, m.height = msg.Width, msg.Height
	}
	idx := v.focusIdx; var cmd tea.Cmd; v.nodes[idx].model, cmd = v.nodes[idx].model.Update(msg)
	if _, isKey := msg.(tea.KeyMsg); isKey { m.runCmd(v.nodes[idx].conf.ID, msg) }
	for i := 0; i < 3; i++ {
		changed := false; for _, n := range v.nodes {
			val := n.model.Value(); if val != n.lastValue {
				n.lastValue = val; changed = true; for _, other := range v.nodes { if other.conf.Watches == n.conf.ID { m.runCmd(other.conf.ID, nil) } }
				m.runCmd(n.conf.ID, nil)
			}
		}
		if !changed { break }
	}
	return m, cmd
}

func (m *model) View() string {
	if m.width == 0 { return "Calculating..." }; v := m.views[m.activeIdx]; var tabs []string; m.tabWidths = make([]int, len(m.views))
	for i, vi := range m.views {
		title := vi.conf.Title; if title == "" { title = vi.conf.ID }; label := title
		if m.topConf.ShowShortcuts && vi.conf.Shortcut != "" { label = fmt.Sprintf("[%s] %s", vi.conf.Shortcut, title) }
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.activeIdx { style = style.Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Bold(true) } else { style = style.Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255")) }
		rendered := style.Render(label); m.tabWidths[i] = lipgloss.Width(rendered); tabs = append(tabs, rendered)
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...); contentH := m.height - 2; if contentH < 0 { contentH = 0 }
	content := m.renderRecursive(v.conf, m.width, contentH, 0, 1)
	statusStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("245")).Width(m.width)
	statusText := fmt.Sprintf(" VIEW: %s | SIZE: %dx%d | Tab: Cycle | Esc: Back", v.conf.ID, m.width, m.height)
	return tabBar + "\n" + content + "\n" + statusStyle.Render(statusText)
}

func (m *model) renderRecursive(conf Config, w, h, offX, offY int) string {
	v := m.views[m.activeIdx]
	if len(conf.Items) == 0 {
		var target *node; for _, n := range v.nodes { if n.conf.ID == conf.ID { target = n; break } }
		if target == nil { return "ERR: " + conf.ID }; target.x, target.y, target.w, target.h = offX, offY, w, h
		if !target.model.Focusable() { target.model.SetSize(w, h); return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).MaxWidth(w).Align(lipgloss.Top, lipgloss.Left).Render(target.model.View()) }
		hasTitle := conf.Title != ""; boxH := h; if hasTitle { boxH = h - 1 }; if boxH < 3 { boxH = 3 }
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Width(w - 2).Height(boxH - 2)
		if v.nodes[v.focusIdx].conf.ID == conf.ID { style = style.Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("205")) }
		target.model.SetSize(w-2, boxH-2); renderedBox := style.Render(target.model.View())
		if !hasTitle { return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).MaxWidth(w).Render(renderedBox) }
		title := lipgloss.NewStyle().Bold(true).MaxWidth(w).Render(conf.Title)
		return lipgloss.NewStyle().Width(w).Height(h).MaxHeight(h).MaxWidth(w).Render(lipgloss.JoinVertical(lipgloss.Left, title, renderedBox))
	}
	var children []string; curX, curY := offX, offY; usableH := h; if conf.Direction != "horizontal" { usableH = h - (len(conf.Items) - 1); if usableH < 0 { usableH = 0 } }
	remainingW, remainingH := w, usableH
	for i, child := range conf.Items {
		childW, childH := w, usableH
		if conf.Direction == "horizontal" {
			if i == len(conf.Items)-1 { childW = remainingW } else { childW = (w * child.WidthPC) / 100 }
			children = append(children, m.renderRecursive(child, childW, h, curX, curY)); curX += childW; remainingW -= childW
		} else {
			if i == len(conf.Items)-1 { childH = remainingH } else { childH = (usableH * child.HeightPC) / 100 }
			children = append(children, m.renderRecursive(child, w, childH, curX, curY)); curY += childH + 1; remainingH -= childH
		}
	}
	if conf.Direction == "horizontal" { return lipgloss.JoinHorizontal(lipgloss.Top, children...) }
	return strings.Join(children, "\n")
}

func watchConfig(path string, p *tea.Program) {
	var lastMod time.Time
	for {
		time.Sleep(1 * time.Second); info, err := os.Stat(path); if err != nil { continue }
		if lastMod.IsZero() { lastMod = info.ModTime(); continue }
		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime(); file, _ := os.ReadFile(path); var top TopLevelConfig
			if yaml.Unmarshal(file, &top) != nil { var root Config; if yaml.Unmarshal(file, &root) == nil { top.Views = []Config{root} } }
			if len(top.Views) > 0 { p.Send(ReloadMsg{Top: top}) }
		}
	}
}

func main() {
	configPath := flag.String("config", "", "Path to YAML"); flag.Parse(); if *configPath == "" { fmt.Println("Usage: -config <path>"); os.Exit(1) }
	file, _ := os.ReadFile(*configPath); var top TopLevelConfig; err := yaml.Unmarshal(file, &top)
	if err != nil || len(top.Views) == 0 { var root Config; yaml.Unmarshal(file, &root); top.Views = []Config{root} }
	m := &model{configPath: *configPath, topConf: top, state: make(map[string]string)}; m.state["cwd"], _ = os.Getwd(); m.buildViews(top)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()); go watchConfig(*configPath, p)
	if _, err := p.Run(); err != nil { fmt.Println(err); os.Exit(1) }
}
