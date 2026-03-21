package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	upperPane = iota
	lowerPane
)

type model struct {
	debounceTicket int
	panes          []textarea.Model
	readOnly       [2]bool
	focusIndex     int
	width          int
	height         int
	lastErr        string
	cmdTo          string
	cmdFrom        string
}

type debounceMsg struct {
	id int
}

func (m *model) runTransform() {
	m.lastErr = ""
	source := m.focusIndex
	target := (m.focusIndex + 1) % 2

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
		m.lastErr = strings.Split(string(out), "\n")[0]
		return
	}
	
	m.panes[target].SetValue(string(out))
}

func (m *model) Init() tea.Cmd { return textarea.Blink }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+l": // Clear All
			m.panes[0].SetValue("")
			m.panes[1].SetValue("")
			m.lastErr = ""
			return m, nil
		case "ctrl+y":
			target := (m.focusIndex + 1) % 2
			content := m.panes[target].Value()
			if content != "" {
				clipboard.WriteAll(content)
				m.lastErr = "Copied to clipboard!"
			}
			return m, nil
		case "tab":
			m.panes[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex + 1) % 2
			cmds = append(cmds, m.panes[m.focusIndex].Focus())
			return m, tea.Batch(cmds...)
		}

	case debounceMsg:
		if msg.id == m.debounceTicket {
			m.runTransform()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		paneHeight := (msg.Height / 2) - 5
		for i := range m.panes {
			m.panes[i].SetWidth(msg.Width - 6)
			m.panes[i].SetHeight(paneHeight)
		}
	}

	passThrough := true
	if m.readOnly[m.focusIndex] {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "up", "down", "left", "right", "pgup", "pgdown", "home", "end":
				passThrough = true
			default:
				passThrough = false
			}
		}
	}

	if passThrough {
		var cmd tea.Cmd
		m.panes[m.focusIndex], cmd = m.panes[m.focusIndex].Update(msg)
		cmds = append(cmds, cmd)
	}

	// Trigger debounce only for writable panes
	if _, ok := msg.(tea.KeyMsg); ok && !m.readOnly[m.focusIndex] {
		// If input is empty, clear output instantly
		if m.panes[m.focusIndex].Value() == "" {
			target := (m.focusIndex + 1) % 2
			m.panes[target].SetValue("")
			m.lastErr = ""
		} else {
			m.debounceTicket++
			currentTicket := m.debounceTicket
			debounceCmd := tea.Tick(250*time.Millisecond, func(_ time.Time) tea.Msg {
				return debounceMsg{id: currentTicket}
			})
			cmds = append(cmds, debounceCmd)
		}
	}

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

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" TAB: switch • CTRL+Y: copy result • CTRL+L: clear • CTRL+C: quit")
	if strings.Contains(m.lastErr, "clipboard") {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render(" ✓ " + m.lastErr)
	} else if m.lastErr != "" {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render(" !! " + m.lastErr)
	}

	return fmt.Sprintf("\n %s\n%s\n\n %s\n%s\n\n%s",
		lipgloss.NewStyle().Bold(true).Render("UPPER (FROM: "+m.cmdFrom+")"), views[0],
		lipgloss.NewStyle().Bold(true).Render("LOWER (TO: "+m.cmdTo+")"), views[1],
		help)
}

func main() {
	to := flag.String("to", "", "Command to transform lower to upper")
	from := flag.String("from", "", "Command to transform upper to lower")
	flag.Parse()

	if *to == "" && *from == "" {
		*to = "base64"
		*from = "base64 -d"
	}

	up := textarea.New()
	lp := textarea.New()

	var initialFocus int
	var ro [2]bool

	if *to != "" && *from == "" {
		ro[upperPane] = true
		up.Placeholder = "Output will appear here..."
		initialFocus = lowerPane
		lp.Focus()
	} else if *from != "" && *to == "" {
		ro[lowerPane] = true
		lp.Placeholder = "Output will appear here..."
		initialFocus = upperPane
		up.Focus()
	} else {
		initialFocus = lowerPane
		lp.Focus()
	}

	m := model{
		panes:      []textarea.Model{up, lp},
		readOnly:   ro,
		cmdTo:      *to,
		cmdFrom:    *from,
		focusIndex: initialFocus,
	}

	if _, err := tea.NewProgram(&m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
	}
}
