package main

// View represents a single step in the TUI chain, executing an external binary.
type View struct {
	Component string            `json:"component"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	OnSuccess string            `json:"on_success"` // e.g., "view:next_id" or "exit"
}

// TuikConfig represents the root JSON structure.
type TuikConfig struct {
	Main   string            `json:"main"`
	Styles map[string]string `json:"styles"`
	Views  map[string]View   `json:"views"`
}
