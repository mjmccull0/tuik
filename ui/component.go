package ui

import (
	"encoding/json"
)

// TextValue handles the "string or object" shorthand for text
type TextValue struct {
	Value string      `json:"value"`
	Style StyleConfig `json:"style"`
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

// ListItem represents an entry in a list
type ListItem struct {
	Label    string `json:"label"`
	Selected bool
}

// Component is our "Universal Node"
type Component struct {
	Type        string      `json:"type"`
	Text        TextValue   `json:"text"`
	Style       StyleConfig `json:"style"`
	Children    []Component `json:"children,omitempty"`
	Items       []ListItem  `json:"items,omitempty"`
	MultiSelect bool        `json:"multi-select"`
}

// UnmarshalJSON implements the "Smart Defaulting" logic
func (c *Component) UnmarshalJSON(data []byte) error {
	type shadow Component
	var s shadow
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = Component(s)

	// Infer Type if missing
	if c.Type == "" {
		var raw map[string]json.RawMessage
		json.Unmarshal(data, &raw)
		if _, ok := raw["text"]; ok {
			c.Type = "text"
		} else if _, ok := raw["items"]; ok {
			c.Type = "list"
		} else if _, ok := raw["children"]; ok {
			c.Type = "container"
		}
	}
	return nil
}
