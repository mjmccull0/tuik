package main

// View represents a single step in the TUI chain, executing an external binary.
type View struct {
	Component string            `json:"component"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	StateSet  map[string]any    `json:"state.set,omitempty"`
	Handler   Handler           `json:"handler,omitempty"`
}

// TuikConfig represents the root JSON structure.
type TuikConfig struct {
	Main   string            `json:"main"`
	Styles map[string]string `json:"styles"`
	Views  map[string]View   `json:"views"`
}

type Transition struct {
	Target   string            `json:"target"`
	SetState map[string]string `json:"set_state"` // Sets "params" for next view
}
