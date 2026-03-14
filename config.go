package main


type StateHooks struct {
	Get string `json:"get"`
	Set string `json:"set"`
}

type Config struct {
	State StateHooks  `json:"state"`
}

// View represents a single step in the TUI chain, executing an external binary.
type View struct {
	Component string            `json:"component"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	Config    Config            `json:"config,omitempty"`
	StateGet  map[string]string `json:"state.get,omitempty"`
	StateSet  map[string]any    `json:"state.set,omitempty"`
	Handler   Handler           `json:"handler,omitempty"`
}

// Tuik represents the root JSON structure.
type Tuik struct {
	Main   string              `json:"main"`
	Styles   map[string]string `json:"styles"`
	Views    map[string]View   `json:"views"`
	Config   Config            `json:"config,omitempty"`
	StateSet map[string]any    `json:"state.set,omitempty"`
}

type Transition struct {
	Target   string            `json:"target"`
	SetState map[string]string `json:"state.set"` // Sets "params" for next view
}
