package ui

import "github.com/charmbracelet/lipgloss"

// StyleConfig maps our kebab-case JSON to Go fields
type StyleConfig struct {
	Color           string `json:"color"`
	BackgroundColor string `json:"background-color"`
	Bold            bool   `json:"bold"`
	Italic          bool   `json:"italic"`
	Underline       bool   `json:"underline"`
	Padding         int    `json:"padding"`
	MarginTop       int    `json:"margin-top"`
	MarginBottom    int    `json:"margin-bottom"`
}

// StyleContext carries inherited traits (like colors) down the tree
type StyleContext struct {
	Foreground string
	Background string
}

// ToLipgloss converts our config into a reusable Lipgloss style
func (s StyleConfig) ToLipgloss() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(s.Bold).
		Italic(s.Italic).
		Underline(s.Underline).
		Padding(s.Padding).
		MarginTop(s.MarginTop).
		MarginBottom(s.MarginBottom)
}
