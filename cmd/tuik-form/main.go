package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net" // Added for net.ParseIP
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Validator func(string) error

func parseRules(ruleStr string) []Validator {
	var v []Validator
	if ruleStr == "" {
		return v
	}

	segments := strings.Split(ruleStr, "|")
	for _, s := range segments {
		s = strings.TrimSpace(s)
		switch {
		case s == "required":
			v = append(v, func(val string) error {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("field is required")
				}
				return nil
			})

		case s == "ip":
			v = append(v, func(val string) error {
				if net.ParseIP(strings.TrimSpace(val)) == nil {
					return fmt.Errorf("invalid IP address (v4 or v6)")
				}
				return nil
			})

		case s == "ipv4":
			re := regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}$`)
			v = append(v, func(val string) error {
				if !re.MatchString(strings.TrimSpace(val)) {
					return fmt.Errorf("invalid IPv4 address")
				}
				return nil
			})

		case s == "ipv6":
			re := regexp.MustCompile(`^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$`)
			v = append(v, func(val string) error {
				if !re.MatchString(strings.TrimSpace(val)) {
					return fmt.Errorf("invalid IPv6 address")
				}
				return nil
			})

		case s == "host":
			re := regexp.MustCompile(`^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*(:\d{1,5})?$`)
			v = append(v, func(val string) error {
				val = strings.TrimSpace(val)
				if val == "localhost" || strings.HasPrefix(val, "localhost:") {
					return nil
				}
				if !re.MatchString(val) {
					return fmt.Errorf("invalid host (e.g., example.com:8080)")
				}
				return nil
			})

		case s == "email":
			re := regexp.MustCompile(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`)
			v = append(v, func(val string) error {
				if !re.MatchString(strings.ToLower(val)) {
					return fmt.Errorf("invalid email address")
				}
				return nil
			})

		case s == "url":
			re := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
			v = append(v, func(val string) error {
				if !re.MatchString(val) {
					return fmt.Errorf("invalid URL (http/https required)")
				}
				return nil
			})

		case strings.HasPrefix(s, "min="):
			limitStr := strings.TrimPrefix(s, "min=")
			if limit, err := strconv.Atoi(limitStr); err == nil {
				v = append(v, func(val string) error {
					if len(val) < limit {
						return fmt.Errorf("too short (min %d)", limit)
					}
					return nil
				})
			}

		case strings.HasPrefix(s, "regex{") && strings.HasSuffix(s, "}"):
			pattern := s[6 : len(s)-1]
			if re, err := regexp.Compile(pattern); err == nil {
				v = append(v, func(val string) error {
					if !re.MatchString(val) {
						return fmt.Errorf("invalid format")
					}
					return nil
				})
			}
		}
	}
	return v
}

func parseOptions(s string) map[string]string {
	opts := make(map[string]string)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			opts[kv[0]] = kv[1]
		}
	}
	return opts
}

// --- FIELD INTERFACE ---
type Field interface {
	Update(tea.Msg) (Field, tea.Cmd)
	View() string
	Focus() tea.Cmd
	Blur()
	Value() string
	Validate() error
	SetError(error)
}

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Italic(true)

// --- INPUT FIELD ---
type inputField struct {
	model textinput.Model
	err   error
	rules []Validator
}

func (f *inputField) Validate() error {
	for _, check := range f.rules {
		if err := check(f.model.Value()); err != nil {
			return err
		}
	}
	return nil
}
func (f *inputField) SetError(err error) { f.err = err }
func (f *inputField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	oldValue := f.model.Value()
	f.model, cmd = f.model.Update(msg)
	if f.model.Value() != oldValue {
		f.err = nil
	}
	return f, cmd
}
func (f *inputField) View() string {
	s := f.model.View()
	if f.err != nil {
		s += "\n" + errStyle.Render("  !! "+f.err.Error())
	}
	return s
}
func (f *inputField) Focus() tea.Cmd { return f.model.Focus() }
func (f *inputField) Blur()          { f.model.Blur() }
func (f *inputField) Value() string  { return f.model.Value() }

// --- FILE FIELD ---
type fileField struct {
	model   filepicker.Model
	focused bool
	path    string
	err     error
}

func (f *fileField) Validate() error {
	if f.path == "" {
		return fmt.Errorf("a file must be selected")
	}
	return nil
}
func (f *fileField) SetError(err error) { f.err = err }
func (f *fileField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	var cmd tea.Cmd
	f.model, cmd = f.model.Update(msg)
	if didSelect, path := f.model.DidSelectFile(msg); didSelect {
		f.path = path
		f.err = nil
	}
	return f, cmd
}
func (f *fileField) View() string {
	var s strings.Builder
	if f.path != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("Selected: "+f.path) + "\n")
	} else if !f.focused {
		s.WriteString(f.model.Styles.Selected.Render("[ No file selected ]"))
	}
	if f.focused {
		f.model.Height = 15
		s.WriteString(lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Render(f.model.View()))
	}
	if f.err != nil {
		s.WriteString("\n" + errStyle.Render("  !! "+f.err.Error()))
	}
	return s.String()
}
func (f *fileField) Focus() tea.Cmd { f.focused = true; return f.model.Init() }
func (f *fileField) Blur()          { f.focused = false }
func (f *fileField) Value() string  { return f.path }

// --- ADDITIONAL FIELDS (Simplified for main loop) ---

type areaField struct {
	model textarea.Model
	err   error
}

func (f *areaField) Validate() error     { return nil }
func (f *areaField) SetError(err error) { f.err = err }
func (f *areaField) Update(msg tea.Msg) (Field, tea.Cmd) {
	var cmd tea.Cmd
	f.model, cmd = f.model.Update(msg)
	return f, cmd
}
func (f *areaField) View() string {
	s := f.model.View()
	if f.err != nil {
		s += "\n" + errStyle.Render("  !! "+f.err.Error())
	}
	return s
}
func (f *areaField) Focus() tea.Cmd { return f.model.Focus() }
func (f *areaField) Blur()          { f.model.Blur() }
func (f *areaField) Value() string  { return f.model.Value() }

type multiField struct {
	options  []string
	selected map[int]bool
	cursor   int
	focused  bool
	err      error
}

func (f *multiField) Validate() error     { return nil }
func (f *multiField) SetError(err error) { f.err = err }
func (f *multiField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "up", "k":
			if f.cursor > 0 {
				f.cursor--
			}
		case "down", "j":
			if f.cursor < len(f.options)-1 {
				f.cursor++
			}
		case " ":
			f.selected[f.cursor] = !f.selected[f.cursor]
		}
	}
	return f, nil
}
func (f *multiField) View() string {
	var s strings.Builder
	for i, opt := range f.options {
		cursor := "  "
		if i == f.cursor {
			cursor = "> "
		}
		checked := "[ ]"
		if f.selected[i] {
			checked = "[x]"
		}
		line := fmt.Sprintf("%s%s %s", cursor, checked, opt)
		if f.focused && i == f.cursor {
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(line) + "\n")
		} else {
			s.WriteString(line + "\n")
		}
	}
	if f.err != nil {
		s.WriteString(errStyle.Render("  !! " + f.err.Error()))
	}
	return s.String()
}
func (f *multiField) Focus() tea.Cmd { f.focused = true; return nil }
func (f *multiField) Blur()          { f.focused = false }
func (f *multiField) Value() string {
	var res []string
	for i, opt := range f.options {
		if f.selected[i] {
			res = append(res, opt)
		}
	}
	return strings.Join(res, ",")
}

type choiceField struct {
	options []string
	index   int
	focused bool
	err     error
}

func (f *choiceField) Validate() error     { return nil }
func (f *choiceField) SetError(err error) { f.err = err }
func (f *choiceField) Update(msg tea.Msg) (Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "up", "k":
			if f.index > 0 {
				f.index--
			}
		case "down", "j":
			if f.index < len(f.options)-1 {
				f.index++
			}
		}
	}
	return f, nil
}
func (f *choiceField) View() string {
	var s strings.Builder
	for i, opt := range f.options {
		if i == f.index {
			s.WriteString("> " + opt + "\n")
		} else {
			s.WriteString("  " + opt + "\n")
		}
	}
	if f.err != nil {
		s.WriteString(errStyle.Render("  !! " + f.err.Error()))
	}
	return s.String()
}
func (f *choiceField) Focus() tea.Cmd { f.focused = true; return nil }
func (f *choiceField) Blur()          { f.focused = false }
func (f *choiceField) Value() string  { return f.options[f.index] }

type submitButton struct {
	label   string
	focused bool
}

func (f *submitButton) Validate() error                { return nil }
func (f *submitButton) SetError(err error)            {}
func (f *submitButton) Update(msg tea.Msg) (Field, tea.Cmd) { return f, nil }
func (f *submitButton) View() string {
	buttonStyle := lipgloss.NewStyle().Padding(0, 3).MarginTop(1).Background(lipgloss.Color("237")).Foreground(lipgloss.Color("250"))
	if f.focused {
		buttonStyle = buttonStyle.Background(lipgloss.Color("205")).Foreground(lipgloss.Color("255")).Bold(true)
	}
	return buttonStyle.Render(f.label)
}
func (f *submitButton) Focus() tea.Cmd { f.focused = true; return nil }
func (f *submitButton) Blur()          { f.focused = false }
func (f *submitButton) Value() string  { return "pressed" }

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
	var cmd tea.Cmd
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
			if fp, ok := m.fields[m.focusIndex].(*fileField); ok {
				if didSelect, _ := fp.model.DidSelectFile(msg); didSelect {
					return m, nil
				}
			}
			if m.focusIndex == len(m.fields)-1 {
				hasErrors, firstErrorIndex := false, -1
				for i, f := range m.fields {
					if err := f.Validate(); err != nil {
						f.SetError(err)
						hasErrors = true
						if firstErrorIndex == -1 {
							firstErrorIndex = i
						}
					} else {
						f.SetError(nil)
					}
				}
				if hasErrors {
					m.fields[m.focusIndex].Blur()
					m.focusIndex = firstErrorIndex
					return m, m.fields[m.focusIndex].Focus()
				}
				m.submitted = true
				return m, tea.Quit
			}
		}
	}
	m.fields[m.focusIndex], cmd = m.fields[m.focusIndex].Update(msg)
	return m, cmd
}

func (m formModel) View() string {
	var s strings.Builder
	for i, f := range m.fields {
		style := lipgloss.NewStyle().Bold(true)
		if i == m.focusIndex {
			style = style.Foreground(lipgloss.Color("205"))
		}
		s.WriteString(style.Render(m.labels[i]) + "\n")
		s.WriteString(f.View() + "\n\n")
	}
	return s.String()
}

type fieldArgs []string

func (i *fieldArgs) String() string     { return "" }
func (i *fieldArgs) Set(v string) error { *i = append(*i, v); return nil }

func main() {
	var fArgs fieldArgs
	flag.Var(&fArgs, "f", "key:type:label[:options_map]")
	flag.Parse()

	m := formModel{}
	for _, arg := range fArgs {
		p := strings.SplitN(arg, ":", 4)
		if len(p) < 3 {
			continue
		}

		key, fType, label := p[0], p[1], p[2]

		var opts map[string]string
		if len(p) > 3 {
			opts = parseOptions(p[3])
		} else {
			opts = make(map[string]string)
		}

		placeholder := opts["placeholder"]
		initial := opts["initial"]
		rules := opts["rules"]
		rawOptions := opts["options"]

		m.keys, m.labels = append(m.keys, key), append(m.labels, label)

		switch fType {
		case "input", "secret":
			ti := textinput.New()
			ti.Placeholder = placeholder
			ti.SetValue(initial)
			if fType == "secret" {
				ti.EchoMode = textinput.EchoPassword
			}
			m.fields = append(m.fields, &inputField{
				model: ti,
				rules: parseRules(rules),
			})

		case "area":
			ta := textarea.New()
			ta.Placeholder = placeholder
			ta.SetValue(initial)
			m.fields = append(m.fields, &areaField{model: ta})

		case "multi":
			splitOpts := strings.Split(rawOptions, ",")
			m.fields = append(m.fields, &multiField{options: splitOpts, selected: make(map[int]bool)})

		case "choice":
			splitOpts := strings.Split(rawOptions, ",")
			m.fields = append(m.fields, &choiceField{options: splitOpts})

		case "confirm":
			m.fields = append(m.fields, &submitButton{label: label})

		case "file":
			fp := filepicker.New()
			fp.AllowedTypes = []string{".go", ".md", ".json"}
			fp.CurrentDirectory, _ = os.Getwd()
			m.fields = append(m.fields, &fileField{model: fp})
		}
	}

	if len(m.fields) == 0 {
		return
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, _ := p.Run()

	if fm, ok := final.(formModel); ok && fm.submitted {
		res := make(map[string]string)
		for i, f := range fm.fields {
			res[fm.keys[i]] = f.Value()
		}
		json.NewEncoder(os.Stdout).Encode(res)
	}
}
