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
	cfg          TuikConfig
	ctx          Context
	activeViewID string
}

type processFinishedMsg struct {
	err error
}

func (model *model) Init() tea.Cmd {
	return model.runActiveView()
}



func (model *model) runActiveView() tea.Cmd {
	v, ok := model.cfg.Views[model.activeViewID]
	if !ok {
		return tea.Quit
	}

	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	env := os.Environ()
	for k, val := range v.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, model.ctx.Resolve(val)))
	}

	resolvedArgs := make([]string, len(v.Args))
	for i, arg := range v.Args {
		val := model.ctx.Resolve(arg)
		if strings.Contains(val, " ") {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

	cmdString := v.Component + " " + strings.Join(resolvedArgs, " ")

	if v.Component == "gum" {
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

func (model *model) onSuccess(onSuccess map[string]interface{}) {
	// Apply nested set_state for this transition
	if newState, ok := onSuccess["set_state"].(map[string]interface{}); ok {
		for k, val := range newState {
			model.setState(k, val)
		}
	}
}

func (model *model) setState(key string, value any) {
    // We use 'any' (alias for interface{}) so we can pass numbers, strings, etc.
    // We resolve the value immediately so the stored state is always "clean"
    resolvedValue := model.ctx.Resolve(fmt.Sprintf("%v", value))
    model.ctx.Data[key] = resolvedValue
}

func (model *model) updateState(view View) {
	// 1. Capture output into state
	if out, err := os.ReadFile(tempOutputFile); err == nil {
		val := strings.TrimSpace(string(out))
		model.ctx.Data["last_output"] = val
		if view.SetState != "" {
			model.ctx.Data[view.SetState] = val
		}
		_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
	}
}

func (model *model) navigate (target string) (tea.Model, tea.Cmd) {
	if target == "exit" || target == "" {
		return model, tea.Quit
	}

	if strings.HasPrefix(target, "view:") {
		model.activeViewID = strings.TrimPrefix(target, "view:")

		return model, model.runActiveView()
	}

	return model, tea.Quit
}

func (m *model) transition(v View) (tea.Model, tea.Cmd) {
	// 1. Simple String Case (The easiest "exit")
	if target, ok := v.OnSuccess.(string); ok {
		return m.navigate(target)
	}

	// 2. Map Case: If it's NOT a map, we don't know what to do. Exit.
	outcome, ok := v.OnSuccess.(map[string]any)
	if !ok {
		return m.navigate("exit")
	}

	// 3. Resolve "What is the next view?" 
	// We check for a direct "target" first, then fall back to menu selection.
	rawTarget := outcome["target"]
	if rawTarget == nil {
		choice := m.ctx.Data["last_output"]
		rawTarget = outcome[choice]
	}

	// 4. If we still have nothing, we're done.
	if rawTarget == nil {
		return m.navigate("exit")
	}

	// 5. Handle the result (String vs Map)
	// Notice we don't nest these; they are mutually exclusive.
	if target, ok := rawTarget.(string); ok {
		return m.navigate(target)
	}

	if targetMap, ok := rawTarget.(map[string]any); ok {
		// Set state if it exists
		m.onSuccess(targetMap)
		
		// Navigate to the target inside the map
		if next, ok := targetMap["target"].(string); ok {
			return m.navigate(next)
		}
	}

	return m.navigate("exit")
}

func (model *model) capture(msg tea.Msg) (tea.Model, tea.Cmd) {
	finishMsg := msg.(processFinishedMsg)
	
	if finishMsg.err != nil {
		if model.activeViewID == model.cfg.Main {
			return model, tea.Quit
		}

		model.activeViewID = model.cfg.Main
		return model, model.runActiveView()
	}

	view := model.cfg.Views[model.activeViewID]
	model.updateState(view)

	return model.transition(view)
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
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	m := &model{
		cfg:          cfg,
		activeViewID: cfg.Main,
		ctx: Context{
			Data:   make(map[string]string),
			Styles: cfg.Styles,
		},
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
