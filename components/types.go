package components 

import (
	"strings"
  "github.com/charmbracelet/lipgloss"
  tea "github.com/charmbracelet/bubbletea"
)


type Context struct {
	// Data holds the variables for template resolution (e.g., "selected_file")
	Data map[string]string
	// Terminal dimensions for responsive components
	Width  int
	Height int
	// Shared styles to maintain visual consistency across views
	Styles map[string]lipgloss.Style
}

// components/types.go

func (c Context) Resolve(input string) string {
    // If there is no placeholder, don't waste time looping
    if !strings.Contains(input, "{{.") {
        return input
    }

    result := input
    for key, value := range c.Data {
        placeholder := "{{." + key + "}}"
        result = strings.ReplaceAll(result, placeholder, value)
    }
    return result
}

type Component interface {
	// Render now takes Context to handle internal template resolution
	Render(ctx Context) string
	
	// Update takes Context so it can potentially modify its behavior 
	// based on the current state of the view
	Update(msg tea.Msg, ctx Context) (Component, tea.Cmd)

	// Focus management
	Focus()
	Blur()
	IsFocusable() bool
	
	// GetID allows the Navigator to know where to save data from this component
	GetID() string
	// GetAction returns the associated on-press/on-select action
	GetAction() string
	GetValue() string
	GetType() string
}

type View struct {
	ID         string               `json:"id"`
	Style      map[string]interface{} `json:"style"`
	Children   []Component          `json:"children,omitempty"`
	Flow       []View               `json:"flow,omitempty"`       // The Wizard logic
	OnComplete string               `json:"on-complete,omitempty"` // The final command
	Context    map[string]interface{} `json:"context,omitempty"`    // Local variables
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
	ID       string
	Input    any
	OnSelect string
	cursor   int
}

type ListItem struct {
	Text     string `json:"text"`
	OnPress  string `json:"on-press,omitempty"`
}

func (v View) Render(ctx Context) string {
	var b strings.Builder
	for _, child := range v.Children {
		b.WriteString(child.Render(ctx))
		b.WriteString("\n")
	}
	return b.String()
}

func (v View) Update(msg tea.Msg, ctx Context) (View, tea.Cmd) {
	var cmds []tea.Cmd
	for i, child := range v.Children {
		newComp, cmd := child.Update(msg, ctx)
		v.Children[i] = newComp
		cmds = append(cmds, cmd)
	}
	return v, tea.Batch(cmds...)
}
