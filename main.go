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
	activeViewID string
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
	view, ok := model.nav.GetView(model.activeViewID)
	if !ok {
		return tea.Quit
	}

	// PHASE 1: Local Hydration
	// We set these before resolving Env or Args so they can use the values
	for key, val := range view.StateSet {
		resolvedVal := model.nav.Resolve(fmt.Sprintf("%v", val))
		model.nav.Set(key, resolvedVal)
	}

	_ = os.WriteFile(tempOutputFile, []byte(""), 0644)

	// Use the Navigator to resolve variables for Env
	env := os.Environ()
	for k, val := range view.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, model.nav.Resolve(val)))
	}

	// PHASE 2: Hybrid Component + Args logic
	resolvedArgs := make([]string, len(view.Args))
	for i, arg := range view.Args {
		val := model.nav.Resolve(arg)
		// Basic quoting for safety in shell -c
		if strings.Contains(val, " ") || val == "" {
			resolvedArgs[i] = fmt.Sprintf("%q", val)
		} else {
			resolvedArgs[i] = val
		}
	}

	// Construct the command from resolved component and args
	cmdString := model.nav.Resolve(view.Component) + " " + strings.Join(resolvedArgs, " ")
	
	// Ensure gum output is redirected to our temp file
	if strings.HasPrefix(view.Component, "gum") {
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

	// 1. Handle Errors (Preserved: returns to main or quits)
	if msg.err != nil {
		if model.activeViewID == mainId {
			return model, tea.Quit
		}
		model.activeViewID = mainId
		return model, model.runActiveView()
	}

	view, ok := model.nav.GetView(model.activeViewID)
	if !ok {
		return model, tea.Quit
	}

	// 2. Capture Output & Handle Mapping logic
	var lastVal string
	if out, err := os.ReadFile(tempOutputFile); err == nil {
		lastVal = strings.TrimSpace(string(out))
		model.nav.Set("last_output", lastVal)
		
		if view.Handler.Stdout != "" {
			model.nav.Set(view.Handler.Stdout, lastVal)
		}
		_ = os.WriteFile(tempOutputFile, []byte(""), 0644)
	}

	// 3. Decide the Next Target: Pattern Matching vs. Success Fallback
	var nextTargetInput any

	// If the output (e.g., "Search Files") exists as a key in our handler map, use it.
	// This makes your 'hub' view work as written.
	if specificTarget, exists := view.Handler.Mapping[lastVal]; exists {
		nextTargetInput = specificTarget
	} else {
		// Otherwise, fall back to the explicit 'success' key (for views like 'pick_file')
		nextTargetInput = view.Handler.Success
	}

	nextTarget := model.nav.DetermineNext(nextTargetInput)
	return model.navigate(nextTarget)
}

// navigate updates the model's current view pointer
func (model *model) navigate(target string) (tea.Model, tea.Cmd) {
	if target == "exit" || target == "" {
		return model, tea.Quit
	}

	if strings.HasPrefix(target, "view:") {
		model.activeViewID = strings.TrimPrefix(target, "view:")
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

	arg := os.Args[1]

	// THE DISPATCHER	
	// Check if the argument is a subcommand (e.g., 'dual-view' -> 'tuik-dual-view')
	subCommand := "tuik-" + arg
	if lp, err := exec.LookPath(subCommand); err == nil {
		// If found, execute the binary and pass remaining args
		cmd := exec.Command(lp, os.Args[2:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		return // Important: stop here if we ran a subcommand
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var cfg Tuik
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	nav := &Navigator{
		Tuik: cfg,
		Data:   make(map[string]string),
	}

	for k, v := range cfg.StateSet {
		val := fmt.Sprintf("%v", v)

		// Update local memory immediately
		// Purposely choosing not to use nav.Set here to avoid duplicate cmd calls.
    nav.Data[k] = val

		// Persist to skate SYNCHRONOUSLY for the boot phase
    persistCmd := cfg.Config.State.Set
    if persistCmd != "" {
        cmd := strings.ReplaceAll(persistCmd, "{{.key}}", k)
        cmd = strings.ReplaceAll(cmd, "{{.value}}", val)
        // No 'go' keyword here! Wait for the command to finish.
        _ = exec.Command("sh", "-c", cmd).Run()
    }
	}

	model := &model{
		activeViewID: cfg.Main,
		nav:          nav,
	}

	initialView, _ := nav.GetView(model.activeViewID)
	nav.HydrateView(initialView)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
