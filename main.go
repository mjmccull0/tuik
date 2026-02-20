package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	redStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	greenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	orangeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	grayStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	defaultStyle = lipgloss.NewStyle()
)

type StyleConfig struct {
	Color      string `json:"color"`
	Background string `json:"background"`
	Bold       bool   `json:"bold"`
	Italic     bool   `json:"italic"`
}

type menuState struct {
	choices      []item
	cursor       int
	id           string
	activeParent item
}

type item struct {
	Label            string                 `json:"label"`
	ID               string                 `json:"id"`
	Exec             string                 `json:"exec"`
	Source           string                 `json:"source"`
	Selected         bool                   `json:"selected"`
	ItemsLabelFilter string                 `json:"items_label_filter"`
	ItemsLabelColor  map[string]string      `json:"items_label_color"`
	Color            string                 `json:"color"`
	Then             []item                 `json:"then"`
	SelectIf         string                 `json:"select_if"`
	ItemsLabelStyle  map[string]StyleConfig `json:"items_label_style"`
	MultiSelect      bool                   `json:"multi_select"`
	ResolvedStyle    lipgloss.Style
}

type model struct {
	stack        []menuState
	cursor       int
	currentID    string
	choices      []item
	chosen       string
	context      map[string][]string
	activeParent item
}

func runSource(parent item) ([]item, error) {
	cmd := exec.Command("bash", "-c", parent.Source)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var items []item

	for _, line := range lines {
		if line == "" {
			continue
		}

		label := line
		selected := false
		resolvedColor := ""
		var finalStyle lipgloss.Style

		// 1. Legacy Color Support
		for regex, colorName := range parent.ItemsLabelColor {
			if matched, _ := regexp.MatchString(regex, line); matched {
				resolvedColor = colorName
				break
			}
		}

		// 2. Full Style Baking
		for pattern, config := range parent.ItemsLabelStyle {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				style := lipgloss.NewStyle()
				if config.Color != "" {
					style = style.Foreground(lipgloss.Color(config.Color))
				}
				if config.Background != "" {
					style = style.Background(lipgloss.Color(config.Background))
				}
				if config.Bold {
					style = style.Bold(true)
				}
				if config.Italic {
					style = style.Italic(true)
				}
				finalStyle = style
				break
			}
		}

		// 3. Auto-selection
		if parent.SelectIf != "" {
			matched, _ := regexp.MatchString(parent.SelectIf, line)
			selected = matched
		}

		// 4. Filtering
		if parent.ItemsLabelFilter != "" {
			if re, err := regexp.Compile(parent.ItemsLabelFilter); err == nil {
				label = re.ReplaceAllString(line, "")
			}
		}

		label = strings.TrimSpace(label)

		items = append(items, item{
			Label:         label,
			Selected:      selected,
			Color:         resolvedColor,
			ResolvedStyle: finalStyle,
		})
	}
	return items, nil
}

func handleEnter(m model) (tea.Model, tea.Cmd) {
	if len(m.choices) == 0 {
		return m, nil
	}
	current := m.choices[m.cursor]

	if m.currentID != "" {
		m.context[m.currentID] = []string{current.Label}
	}

	nextID := m.currentID
	if current.ID != "" {
		nextID = current.ID
	}

	// Navigation: Source or Sub-menu
	if current.Source != "" || len(current.Then) > 0 {
		m.stack = append(m.stack, menuState{
			choices:      m.choices,
			cursor:       m.cursor,
			id:           m.currentID,
			activeParent: m.activeParent,
		})

		m.activeParent = current
		m.cursor = 0
		m.currentID = nextID

		if current.Source != "" {
			newItems, err := runSource(current)
			if err != nil {
				return m, nil
			}
			m.choices = newItems
		} else {
			m.choices = current.Then
		}
		return m, nil
	}

	// Execution: Template Injection
	if current.Exec != "" {
		var selected []string
		for _, it := range m.choices {
			if it.Selected {
				selected = append(selected, it.Label)
			}
		}
		if len(selected) == 0 {
			selected = append(selected, current.Label)
		}

		fileList := strings.Join(selected, " ")
		finalCmd := strings.ReplaceAll(current.Exec, "{{.files}}", fileList)
		m.chosen = os.ExpandEnv(finalCmd)
		return m, tea.Quit
	}

	return m, nil
}

func handleBack(m model) (tea.Model, tea.Cmd) {
	if len(m.stack) == 0 {
		return m, tea.Quit
	}

	lastIndex := len(m.stack) - 1
	previousState := m.stack[lastIndex]

	m.choices = previousState.choices
	m.cursor = previousState.cursor
	m.currentID = previousState.id
	m.activeParent = previousState.activeParent
	m.stack = m.stack[:lastIndex]

	return m, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "esc", "backspace":
			return handleBack(m)
		case "enter":
			return handleEnter(m)
		case " ":
			if len(m.choices) > 0 && m.activeParent.MultiSelect {
				m.choices[m.cursor].Selected = !m.choices[m.cursor].Selected
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Tuik Menu\n\n"

	for i, choice := range m.choices {
		// 1. FIXED GUTTER: Always 6 characters wide
		// [2 chars for cursor][4 chars for checkbox]
		
		cursorSymbol := "  "
		if m.cursor == i {
			cursorSymbol = "> "
		}

		checkbox := ""
		if m.activeParent.MultiSelect {
			if choice.Selected {
				checkbox = "[x] "
			} else {
				checkbox = "[ ] "
			}
		} else {
			// If not multiselect, add 4 spaces to keep labels aligned with 
			// the items that DO have checkboxes.
			checkbox = "    " 
		}

		// 2. CLEAN THE LABEL: Remove any ghost whitespace from the source command
		cleanLabel := strings.TrimSpace(choice.Label)

		// 3. STYLE LOGIC
		style := choice.ResolvedStyle
		if m.activeParent.MultiSelect {
			if choice.Selected {
				style = greenStyle
			} else if style.GetForeground() == (lipgloss.NoColor{}) {
				if choice.Color == "red" || choice.Color == "orange" {
					style = redStyle
				}
			}
		}

		// Fallback for simple colors
		if style.GetForeground() == (lipgloss.NoColor{}) {
			switch choice.Color {
			case "red":    style = redStyle
			case "green":  style = greenStyle
			case "orange": style = orangeStyle
			case "gray":   style = grayStyle
			}
		}

		// 4. RENDER: The gutter stays outside the style to ensure alignment 
		// even if the style adds padding/margin later.
		gutter := cursorSymbol + checkbox
		s += fmt.Sprintf("%s%s\n", gutter, style.Render(cleanLabel))
	}

	s += "\n(space: toggle, enter: confirm, backspace: back, q: quit)"
	return s
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: tuik <config.json>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var items []item
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Fprintf(os.Stderr, "JSON Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model{
		choices: items,
		context: make(map[string][]string),
	}, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	m := finalModel.(model)
	if m.chosen != "" {
		fmt.Print(m.chosen)
	}
}
