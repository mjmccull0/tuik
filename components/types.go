package components 

import (
	"strings"
  "github.com/charmbracelet/lipgloss"
  tea "github.com/charmbracelet/bubbletea"
	"tuik/utils"
	"regexp"
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

func (c *Context) Resolve(input string) string {
    if c.Data == nil || !strings.Contains(input, "{{.") {
        return input
    }

    // This regex handles both {{.key}} and {{.key:-fallback}}
	  re := regexp.MustCompile(`\{\{\.([a-zA-Z0-9_]+)(?::-(.*?))?\}\}`)

    return re.ReplaceAllStringFunc(input, func(match string) string {
        submatches := re.FindStringSubmatch(match)
        key := submatches[1]
        fallback := submatches[2]

        utils.Log("key: '%s'", key)
        utils.Log("fallback: '%s'", fallback)
        // Use the live data from the pointer receiver
        if val, ok := c.Data[key]; ok && val != "" {
            utils.Log("val: '%s'", val)
            return val
        }
        return fallback
    })
}

type Component interface {
	// Render now takes Context to handle internal template resolution
	Render(ctx *Context) string
	
	// Update takes Context so it can potentially modify its behavior 
	// based on the current state of the view
	Update(msg tea.Msg, ctx *Context) (Component, tea.Cmd)

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
	Main        string           `json:"main"`
	Views       map[string]*View `json:"views"`
	ShellFunctions string
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
	Cursor   int
}

type ListItem struct {
	Text     string `json:"text"`
	OnPress  string `json:"on-press,omitempty"`
}


type ListHydrationMsg struct {
	ID    string
	Items []ListItem
}

func (c *Context) Set(key string, value string) {
    if c.Data == nil {
        c.Data = make(map[string]string)
    }
    c.Data[key] = value
}

func (c *Context) ReplacePlaceholders(input string) string {
  utils.Log("REPLACEMENT INPUT: '%s'", input)
	output := c.Resolve(input)
	utils.Log("REPLACEMENT OUTPUT: '%s'", output)
	return output
}

// Ensure View is a pointer in the navigator map
// type Navigator struct { Views map[string]*View ... }
func (v *View) Update(msg tea.Msg, ctx *Context) (Component, tea.Cmd) {
	var cmds []tea.Cmd
	for i, child := range v.Children {
		// Update each child
		newComp, cmd := child.Update(msg, ctx)
		v.Children[i] = newComp
		cmds = append(cmds, cmd)
	}
	return v, tea.Batch(cmds...) // Return 'v' as the pointer
}

func (v *View) Render(ctx *Context) string {
	var sections []string
	for _, child := range v.Children {
		sections = append(sections, child.Render(ctx))
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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

func (v *View) Blur() {} // Add this!
func (v *View) Focus() {} // Add this too, just in case

// Ensure these exist so View can be treated as a Component
func (v *View) GetID() string     { return v.ID }
func (v *View) GetType() string   { return "view" }
func (v *View) GetAction() string { return "" }
func (v *View) GetValue() string  { return "" }
func (v *View) IsFocusable() bool { return false }
