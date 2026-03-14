package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	// "path/filepath"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var (
	// Example: TUIK_FOCUS_COLOR=5 (Purple)
	focusColor = getEnv("TUIK_FOCUS_COLOR", "62") 
	borderColor = getEnv("TUIK_BORDER_COLOR", "240")

	docStyle = lipgloss.NewStyle().Margin(1, 2)
	
	activeStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(focusColor))

	inactiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(borderColor))
)

type item struct {
	title, desc, path string
}

type editorFinishedMsg struct{ err error }

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type focus int
const (
	focusList focus = iota
	focusViewport
)

type model struct {
	list         list.Model
	viewport     viewport.Model
	focus        focus
	ready        bool
	selectedPath string
	width, height int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "g":
			if m.focus == focusViewport {
				m.viewport.GotoTop()
			}
		case "G":
			if m.focus == focusViewport {
				m.viewport.GotoBottom()
			}
		case "h", "left":
		  m.focus = focusList
		case "l", "right":
		  m.focus = focusViewport
		case "e":
			if i, ok := m.list.SelectedItem().(item); ok {
        editor := os.Getenv("EDITOR")
        if editor == "" {
            editor = "vi" // Fallback
        }

        // Create the command
        c := exec.Command(editor, i.path)
        
        // tea.ExecProcess takes a command and a termination function.
        // It pauses the TUI and gives the editor full control of Stdin/Stdout.
        return m, tea.ExecProcess(c, func(err error) tea.Msg {
            return editorFinishedMsg{err}
        })
    }
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.selectedPath = i.path
				return m, tea.Quit
			}

		}

	// You'll also need to handle the message returned when the editor closes:
	case editorFinishedMsg:
		if msg.err != nil {
				return m, tea.Quit // Or handle error
		}
		// Optional: Refresh the viewport content in case the file changed
		if i, ok := m.list.SelectedItem().(item); ok {
				content, _ := ioutil.ReadFile(i.path)
				m.viewport.SetContent(string(content))
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		// Calculate horizontal space
    // Total width minus document margins and the gap between panes
    hMargin, vMargin := docStyle.GetFrameSize()
    totalAvailWidth := msg.Width - hMargin
		
		// Split screen: 1/3 for list, 2/3 for preview
		listWidth := totalAvailWidth / 3

		// Calculate vertical space
    // Total height minus top/bottom margins and 1 row for your help bar
    availHeight := msg.Height - vMargin - 3 // -3 accounts for borders and footer

		m.list.SetSize(listWidth, availHeight)

	  viewWidth := totalAvailWidth - listWidth - 4	
		if !m.ready {
			m.viewport = viewport.New(viewWidth, availHeight)
			m.ready = true
		} else {
			m.viewport.Width = viewWidth
			m.viewport.Height = availHeight
		}
	}

  // Logic: Send keys to the component that has focus
	if m.focus == focusList {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		// Update preview whenever list selection changes
		if i, ok := m.list.SelectedItem().(item); ok {
			content, _ := ioutil.ReadFile(i.path)
			m.viewport.SetContent(string(content))
		}
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Panes
	// Apply styles based on focus
	var lStyle, vStyle lipgloss.Style
	if m.focus == focusList {
		lStyle = activeStyle.Copy()
		vStyle = inactiveStyle.Copy()
	} else {
		lStyle = inactiveStyle.Copy()
		vStyle = activeStyle.Copy()
	}

	// APPLY the widths to the styles
	// The list.View() already has its own internal padding, 
	// but the border needs to know how wide to be.
	lStyle = lStyle.Width(m.list.Width())
	vStyle = vStyle.Width(m.viewport.Width)

	panes := lipgloss.JoinHorizontal(
			lipgloss.Top,
	lStyle.Render(m.list.View()),
					 vStyle.Render(m.viewport.View()),
	)

	// Global Footer (Outside the panes)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
	  Margin(0, 0, 1, 2)

	help := helpStyle.Render("h/l: switch focus • j/k: scroll • e: edit • enter: select • q: quit")

	// Vertical Stack
	return docStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left, 
			panes, 
			help,
		),
	)
}

func main() {
	files, _ := ioutil.ReadDir(".")
	var items []list.Item
	for _, f := range files {
		if !f.IsDir() {
			items = append(items, item{title: f.Name(), desc: "File", path: f.Name()})
		}
	}

	m := model{
		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
		focus: focusList,
	}
	m.list.Title = "Files"
	m.list.SetShowHelp(false) // Hide list pane help menu

	// Run TUI
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()

	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// 3. Emit JSON Protocol
	if m, ok := finalModel.(model); ok && m.selectedPath != "" {
		output := map[string]string{"selected_path": m.selectedPath}
		jsonBytes, _ := json.Marshal(output)
		// tuik will capture this stdout
		fmt.Println(string(jsonBytes))
	}
}
