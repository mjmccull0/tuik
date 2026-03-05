package main

// View represents a single step in the TUI chain, executing an external binary.
type View struct {
	Component string            `json:"component"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
<<<<<<< Updated upstream
	OnSuccess string            `json:"on_success"` // e.g., "view:next_id" or "exit"
=======
	SetState  string            `json:"set_state"`
	OnSuccess interface{}       `json:"on_success"`
>>>>>>> Stashed changes
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
