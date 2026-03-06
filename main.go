package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const tempOutputFile = "/tmp/tuik_exchange.tmp"

type model struct {
	activeViewId string
	nav          *Navigator
}

type processFinishedMsg struct {
	err error
}

func (model *model) Init() tea.Cmd {
	return model.runActiveView()
}

// runActiveView handles the OS-level execution
func (model *model) runActiveView() tea.Cmd {
	view, ok := model.nav.GetView(model.activeViewId)
	if !ok {
		return tea.Quit
	}

	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// Use the Navigator to resolve variables for Env and Args
	env := os.Environ()
	for k, val := range view.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, model.nav.Resolve(val)))
	}

	resolvedArgs := make([]string, len(view.Args))
	for i, arg := range view.Args {
		val := model.nav.Resolve(arg)
		if strings.Contains(val, " ") {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

	cmdString := view.Component + " " + strings.Join(resolvedArgs, " ")
	if view.Component == "gum" {
		cmdString = fmt.Sprintf("%s > %s", cmdString, tempOutputFile)
	}

	c := exec.Command("sh", "-c", cmdString)
	c.Env = env
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return processFinishedMsg{err: err}
	})
}

// capture handles the transition after a process exits
func (model *model) capture(msg processFinishedMsg) (tea.Model, tea.Cmd) {
	mainId := model.nav.GetMainId()

	// Handle Errors (Canceled or Crashed)
	if msg.err != nil {
		if model.activeViewId == mainId {
			return model, tea.Quit
		}

		model.activeViewId = mainId

		return model, model.runActiveView()
	}

	view, ok := model.nav.GetView(model.activeViewId)

	if !ok {
		// If we somehow lost the view, safety exit
		return model, tea.Quit
	}

	// Capture Output into Navigator State
	if out, err := os.ReadFile(tempOutputFile); err == nil {
		val := strings.TrimSpace(string(out))
		model.nav.Set("last_output", val)
		
		if view.SetState != "" {
			model.nav.Set(view.SetState, val)
		}
		_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
	}

	// 3. Let Navigator decide where to go
	nextTarget := model.nav.DetermineNext(view.OnSuccess)
	return model.navigate(nextTarget)
}

// navigate updates the model's current view pointer
func (model *model) navigate(target string) (tea.Model, tea.Cmd) {
	if target == "exit" || target == "" {
		return model, tea.Quit
	}

	if strings.HasPrefix(target, "view:") {
		model.activeViewId = strings.TrimPrefix(target, "view:")
		return model, model.runActiveView()
	}

	return model, tea.Quit
}

func (model *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case processFinishedMsg:
		return model.capture(msg)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return model, tea.Quit
		}
	}
	return model, nil
}

func (model *model) View() string { return "" }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: tuik <config.json>")
		return
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var cfg TuikConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Initialize the Brain
	nav := &Navigator{
		Config: cfg,
		Data:   make(map[string]string),
		Styles: cfg.Styles,
	}

	// Initialize the Body
	model := &model{
		activeViewId: cfg.Main,
		nav:          nav,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
