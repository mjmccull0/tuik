package main

import (
	"fmt"
	"os"
	"os/exec"
	"flag"
	"strconv"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- TYPES ---

type Theme struct {
	FocusColor  lipgloss.Color
	BorderColor lipgloss.Color
	DocMargin   int
}

type item struct {
	title, desc, path, content string
}

type model struct {
	list          list.Model
	viewport      viewport.Model
	theme         Theme
	focus         focus
	ready         bool
	selectedPath  string
	width, height int
}

// --- HELPERS ---

func getEnv[T string | int](key string, fallback T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	var result any
	switch any(fallback).(type) {
	case string:
		result = value
	case int:
		i, err := strconv.Atoi(value)
		if err != nil {
			return fallback
		}
		result = i
	}
	return result.(T)
}

func LoadTheme() Theme {
	return Theme{
		FocusColor:  lipgloss.Color(getEnv("TUIK_COLOR_FOCUS", "62")),
		BorderColor: lipgloss.Color(getEnv("TUIK_COLOR_BORDER", "240")),
		DocMargin:   getEnv("TUIK_MARGIN", 1),
	}
}

// --- BOILERPLATE ---

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type focus int
const (
	focusList focus = iota
	focusViewport
)

type editorFinishedMsg struct{ err error }

func (m model) Init() tea.Cmd { return nil }

// --- UPDATE & VIEW (Using m.theme) ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "h", "left":
			m.focus = focusList
		case "l", "right":
			m.focus = focusViewport
		case "g":
			if m.focus == focusViewport { m.viewport.GotoTop() }
		case "G":
			if m.focus == focusViewport { m.viewport.GotoBottom() }
		case "e":
			if i, ok := m.list.SelectedItem().(item); ok {
				editor := getEnv("EDITOR", "vi")
				c := exec.Command(editor, i.path)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return editorFinishedMsg{err}
				})
			}
		}

	case tea.WindowSizeMsg:
    m.width, m.height = msg.Width, msg.Height

    hMargin := m.theme.DocMargin * 2
    vMargin := m.theme.DocMargin * 2

    // 1. Calculate available width after document margins
    totalAvailWidth := msg.Width - hMargin
    
    // 2. Fixed size for the list (1/3 of the screen)
    listWidth := totalAvailWidth / 3
    
    // 3. GREEDY math for the viewport:
    // Subtract the list width, and EXACTLY 4 cells for the borders 
    // (2 for the list border, 2 for the viewport border)
    viewWidth := totalAvailWidth - listWidth - 4

    // 4. Handle height (subtract footer and margins)
    availHeight := msg.Height - vMargin - 3

    m.list.SetSize(listWidth, availHeight)
    
    if !m.ready {
        m.viewport = viewport.New(viewWidth, availHeight)
        m.ready = true
    } else {
        m.viewport.Width = viewWidth
        m.viewport.Height = availHeight
    }

	case editorFinishedMsg:
		if i, ok := m.list.SelectedItem().(item); ok {
			content, _ := os.ReadFile(i.path)
			i.content = string(content) // Update the stored content
			m.viewport.SetContent(i.content)
		}
	}

	// Route updates
	if m.focus == focusList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		if i, ok := m.list.SelectedItem().(item); ok {
			m.viewport.SetContent(i.content)
		}
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready { return "\n  Initializing..." }

	baseStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderColor)

	var lStyle, vStyle lipgloss.Style
	if m.focus == focusList {
		lStyle = baseStyle.Copy().BorderForeground(m.theme.FocusColor).Width(m.list.Width())
		vStyle = baseStyle.Copy().Width(m.viewport.Width)
	} else {
		lStyle = baseStyle.Copy().Width(m.list.Width())
		vStyle = baseStyle.Copy().BorderForeground(m.theme.FocusColor).Width(m.viewport.Width)
	}

	panes := lipgloss.JoinHorizontal(lipgloss.Top, lStyle.Render(m.list.View()), vStyle.Render(m.viewport.View()))
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginLeft(2).
		Render("h/l: focus • g/G: top/bottom • e: edit • q: quit")

	return lipgloss.NewStyle().Margin(m.theme.DocMargin).
		Render(lipgloss.JoinVertical(lipgloss.Left, panes, help))
}

func main() {
	titleFlag := flag.String("title", "Navigator", "Pane title")
	flag.Parse()

	targetDir := "."
	if args := flag.Args(); len(args) > 0 { targetDir = args[0] }

	files, _ := os.ReadDir(targetDir)
	var items []list.Item
	for _, f := range files {
		if !f.IsDir() {
			path := filepath.Join(targetDir, f.Name())
			content, _ := os.ReadFile(path)
			items = append(items, item{title: f.Name(), desc: "File", path: path, content: string(content)})
		}
	}

	m := model{
		list:  list.New(items, list.NewDefaultDelegate(), 0, 0),
		focus: focusList,
		theme: LoadTheme(),
	}
	m.list.Title = *titleFlag
	m.list.SetShowHelp(false)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
