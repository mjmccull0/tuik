package parser

import (
	"encoding/json"
	"fmt"
	"tuik/components"

	"github.com/charmbracelet/bubbles/textinput"
)

// ParseConfig converts the raw JSON bytes into a structured component registry.
func ParseConfig(data []byte) (components.Config, error) {
	var raw struct {
		Main  string `json:"main"`
		Views map[string]struct {
			Style    map[string]interface{}   `json:"style"`
			Children []map[string]interface{} `json:"children"`
		} `json:"views"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return components.Config{}, err
	}

	config := components.Config{
		Main:  raw.Main,
		Views: make(map[string]*components.View),
	}

	for viewID, viewData := range raw.Views {
		v := &components.View{ID: viewID}

		for _, childData := range viewData.Children {
			comp := ParseComponent(childData)
			if comp != nil {
				v.Children = append(v.Children, comp)
			}
		}
		config.Views[viewID] = v
	}

	return config, nil
}

func ParseComponent(data map[string]interface{}) components.Component {
	typ, _ := data["type"].(string)

	if typ == "" {
		if _, hasText := data["text"]; hasText {
			typ = "text"
		}
	}

	var base components.Component

	switch typ {
	case "text":
		content, _ := data["content"].(string)
		if content == "" {
			content, _ = data["text"].(string)
		}
		base = components.Text{Content: content}

	case "text-input":
		id, _ := data["id"].(string)
		placeholder, _ := data["placeholder"].(string)
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Focus()
		base = &components.TextInput{
			ID:          id,
			Placeholder: placeholder,
			Model:       ti,
		}

	case "list":
		id, _ := data["id"].(string)
		onSelect, _ := data["on-select"].(string)
		var items []components.ListItem
		if itemsRaw, ok := data["items"].([]interface{}); ok {
			for _, it := range itemsRaw {
				if m, ok := it.(map[string]interface{}); ok {
					txt, _ := m["text"].(string)
					act, _ := m["on-press"].(string)
					items = append(items, components.ListItem{Text: txt, OnPress: act})
				}
			}
		}
		inputData := data["input"]
		if inputData == nil {
			inputData = items
		}
		base = components.List{ID: id, Input: inputData, OnSelect: onSelect}

	case "box":
		id, _ := data["id"].(string)
		vertical, _ := data["vertical"].(bool)
		var children []components.Component
		if childrenRaw, ok := data["children"].([]interface{}); ok {
			for _, childData := range childrenRaw {
				if m, ok := childData.(map[string]interface{}); ok {
					if c := ParseComponent(m); c != nil {
						children = append(children, c)
					}
				}
			}
		}
		// FIX: Return the pointer directly here
		base = &components.Box{
			ID:       id,
			Children: children,
			Vertical: vertical,
		}

	case "button":
		text, _ := data["text"].(string)
		action, _ := data["on-select"].(string)
		if action == "" {
			action, _ = data["on-press"].(string)
		}
		id, _ := data["id"].(string)
		base = &components.Button{
			ID:     id,
			Text:   text,
			Action: action,
		}

	default:
		return nil
	}

	// FIX: Style Wrapping - must return &components.Box
	if styleData, ok := data["style"].(map[string]interface{}); ok {
		return &components.Box{
			ID:       "wrap-" + base.GetID(),
			Children: []components.Component{base},
			Styles:   extractStyles(styleData),
		}
	}

	return base
}

// extractStyles maps a JSON map to the internal StyleConfig struct.
func extractStyles(s map[string]interface{}) components.StyleConfig {
	styles := components.StyleConfig{}

	if val, ok := s["padding"].(float64); ok {
		styles.Padding = int(val)
	}
	if val, ok := s["margin-top"].(float64); ok {
		styles.MarginTop = int(val)
	}
	if val, ok := s["margin-bottom"].(float64); ok {
		styles.MarginBottom = int(val)
	}
	if val, ok := s["border"].(bool); ok {
		styles.Border = val
	}
	if val, ok := s["background-color"].(string); ok {
		styles.BackgroundColor = val
	}

	return styles
}

func validate(cfg *components.Config) error {
	if _, ok := cfg.Views[cfg.Main]; !ok {
		return fmt.Errorf("main view '%s' is not defined in views", cfg.Main)
	}

	for viewID, view := range cfg.Views {
		for _, child := range view.Children {
			if err := validateComponent(child); err != nil {
				return fmt.Errorf("view '%s': %w", viewID, err)
			}
		}
	}
	return nil
}

func validateComponent(c components.Component) error {
	// Logic to ensure components that need IDs have them
	if c.GetID() == "" && c.GetType() != "box" {
		return fmt.Errorf("component of type '%s' is missing a required ID", c.GetType())
	}
	return nil
}
