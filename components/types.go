package components 

import tea "github.com/charmbracelet/bubbletea"

type RenderContext struct {
	IsFocused bool
	Store     map[string]string
}

type Component interface {
	Render(ctx RenderContext) string
	Update(msg tea.Msg) (Component, tea.Cmd)
	IsFocusable() bool
	GetType() string
	GetID() string
	GetValue() string
	GetAction() string
}

type View struct {
	ID       string
	Children []Component
}

type Config struct {
	Main  string           `json:"main"`
	Views map[string]*View `json:"views"`
}

type StyleConfig struct {
	Padding         int    `json:"padding"`
	Margin          int    `json:"margin"`
	MarginTop       int    `json:"margin-top"`
	MarginBottom    int    `json:"margin-bottom"`
	Border          bool   `json:"border"`
	BorderColor     string `json:"border-color"`
	BackgroundColor string `json:"background-color"`
	Align           string `json:"align"`
}

type List struct {
	Items  []ListItem // We'll move ListItem to the lib package
	Cursor int           // Local state!
}

type ListItem struct {
	Text     string `json:"text"`
	OnPress  string `json:"on-press"`
}
