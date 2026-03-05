package main

type View struct {
	Component string            `json:"component"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	OnSuccess string            `json:"on_success"`
}

type TuikConfig struct {
	Main   string           `json:"main"`
	Styles map[string]string `json:"styles"`
	Views  map[string]View   `json:"views"`
}
