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
	  // If we have no data, we can't replace anything
    if c.Data == nil {
        return input 
    }

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
	PaddingTop       int   `json:"padding-top"`
	PaddingRight     int   `json:"padding-right"`
	PaddingBottom    int   `json:"padding-bottom"`
	PaddingLeft      int   `json:"padding-left"`
	Margin          int    `json:"margin"`
	MarginTop       int    `json:"margin-top"`
	MarginRight     int    `json:"margin-right"`
	MarginBottom    int    `json:"margin-bottom"`
	MarginLeft      int    `json:"margin-left"`
	Border          bool   `json:"border"`
	BorderColor     string `json:"border-color"`
	BackgroundColor string `json:"background-color"`
	Align           string `json:"align"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
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
	// 1. Ensure the context has a map initialized
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}

	// 2. Merge View's local Context into the render Context
	// We cast the interface{} to string here
	for k, val := range v.Context {
		if strVal, ok := val.(string); ok {
			ctx.Data[k] = strVal
		}
	}

	var b strings.Builder
	for _, child := range v.Children {
		// 3. Now children receive a context that actually contains the data
		b.WriteString(child.Render(ctx))
		b.WriteString("\n")
	}

	return b.String()
}

func (v View) Update(msg tea.Msg, ctx Context) (View, tea.Cmd) {
	// 1. Merge local context so children have it during updates
	if ctx.Data == nil {
		ctx.Data = make(map[string]string)
	}
	for k, val := range v.Context {
		if strVal, ok := val.(string); ok {
			ctx.Data[k] = strVal
		}
	}

	var cmds []tea.Cmd
	for i, child := range v.Children {
		// 2. Update each child with the populated context
		newChild, cmd := child.Update(msg, ctx)
		v.Children[i] = newChild
		cmds = append(cmds, cmd)
	}
	return v, tea.Batch(cmds...)
}

// ActionMsg is sent when a component (like a Button) is activated
type ActionMsg struct {
	ID     string
	Action string
}

func (s StyleConfig) ToLipgloss() lipgloss.Style {
	style := lipgloss.NewStyle()

	if s.Width > 0 {
		style = style.Width(s.Width)
	}
	if s.PaddingTop > 0 || s.PaddingRight > 0 || s.PaddingBottom > 0 || s.PaddingLeft > 0 {
		style = style.Padding(s.PaddingTop, s.PaddingRight, s.PaddingBottom, s.PaddingLeft)
	}
	if s.MarginTop > 0 || s.MarginRight > 0 || s.MarginBottom > 0 || s.MarginLeft > 0 {
		style = style.Margin(s.MarginTop, s.MarginRight, s.MarginBottom, s.MarginLeft)
	}
    
    // Add border logic if you want the Boxes themselves to have borders
	if s.Border {
		style = style.Border(lipgloss.NormalBorder())
	}

	return style
}
