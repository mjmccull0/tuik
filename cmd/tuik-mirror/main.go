package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	upperPane = iota
	lowerPane
)

type model struct {
	panes      []textarea.Model
	focusIndex int
	width      int
	height     int
	lastErr    string
	cmdTo      string
	cmdFrom    string
}

func (m *model) runTransform() {
	m.lastErr = ""
	source := m.focusIndex
	target := (m.focusIndex + 1) % 2
	
	// If the target has no command to receive this data, stop.
	if (source == lowerPane && m.cmdTo == "") || (source == upperPane && m.cmdFrom == "") {
		return
	}

	input := m.panes[source].Value()
	if input == "" {
		m.panes[target].SetValue("")
		return
	}

	shellCmd := m.cmdTo
	if source == upperPane {
		shellCmd = m.cmdFrom
	}

	cmd := exec.Command("sh", "-c", shellCmd)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()

	if err != nil {
		// Slice the error message to keep it concise in the TUI
		m.lastErr = strings.Split(string(out), "\n")[0] 
		return
	}
	m.panes[target].SetValue(strings.TrimSpace(string(out)))
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
			// Only allow tabbing if both commands exist
			if m.cmdTo != "" && m.cmdFrom != "" {
					m.panes[m.focusIndex].Blur()
					m.focusIndex = (m.focusIndex + 1) % 2
					cmds = append(cmds, m.panes[m.focusIndex].Focus())
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.panes[0].SetWidth(msg.Width - 6)
		m.panes[1].SetWidth(msg.Width - 6)
	}

	var cmd tea.Cmd
	m.panes[m.focusIndex], cmd = m.panes[m.focusIndex].Update(msg)
	cmds = append(cmds, cmd)

	// Trigger transformation
	m.runTransform()

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	activeStyle := lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	
	var views [2]string
	for i := range m.panes {
		if i == m.focusIndex {
			views[i] = activeStyle.Render(m.panes[i].View())
		} else {
			style := inactiveStyle
			if (i == upperPane && m.cmdTo == "") || (i == lowerPane && m.cmdFrom == "") {
				style = style.Foreground(lipgloss.Color("237")) 
			}

			views[i] = style.Render(m.panes[i].View())
		}
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" TAB: swap focus • CTRL+C: quit")
	if m.lastErr != "" {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render(" !! " + m.lastErr)
	}

	return fmt.Sprintf("\n %s\n%s\n\n %s\n%s\n\n%s", 
		lipgloss.NewStyle().Bold(true).Render("UPPER (CMD: "+m.cmdFrom+")"), views[0],
		lipgloss.NewStyle().Bold(true).Render("LOWER (CMD: "+m.cmdTo+")"), views[1],
		help)
}

func main() {
	to := flag.String("to", "", "Command to transform lower to upper")
	from := flag.String("from", "", "Command to transform upper to lower")
	flag.Parse()

	// Fallback to base64 only if BOTH are empty
	if *to == "" && *from == "" {
		*to = "base64"
		*from = "base64 -d"
	}

	up := textarea.New()
	lp := textarea.New()

	// Setup Unidirectional Flow
	var initialFocus int
	if *to != "" && *from == "" {
		// Only 'to' exists: Lower is master, Upper is Read-Only
		up.Placeholder = "(Output Only)"
		initialFocus = lowerPane
		lp.Focus()
	} else if *from != "" && *to == "" {
		// Only 'from' exists: Upper is master, Lower is Read-Only
		lp.Placeholder = "(Output Only)"
		initialFocus = upperPane
		up.Focus()
	} else {
		// Bi-directional
		initialFocus = upperPane
		up.Focus()
	}

	m := model{
		panes:   []textarea.Model{up, lp},
		cmdTo:   *to,
		cmdFrom: *from,
		focusIndex: initialFocus,
	}

	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
	}
}
