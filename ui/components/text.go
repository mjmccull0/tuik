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
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Value = s
		return nil
	}
	type alias TextValue
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*t = TextValue(obj)
	return nil
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
