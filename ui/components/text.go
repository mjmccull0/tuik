package components

import (
	"encoding/json"
	"github.com/charmbracelet/lipgloss"
)

type TextValue struct {
	Value string      `json:"value"`
	Style struct {
		Color     string `json:"color"`
		Bold      bool   `json:"bold"`
		Underline bool   `json:"underline"`
	} `json:"style"`
}

func (t *TextValue) UnmarshalJSON(data []byte) error {
	// 1. Try to unmarshal as a plain string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Value = s
		return nil
	}

	// 2. Fallback to unmarshaling as a full object
	type alias TextValue
	return json.Unmarshal(data, (*alias)(t))
}

func RenderText(t TextValue) string {
	style := lipgloss.NewStyle().
		Bold(t.Style.Bold).
		Underline(t.Style.Underline)

	if t.Style.Color != "" {
		style = style.Foreground(lipgloss.Color(t.Style.Color))
	}

	return style.Render(t.Value)
}
