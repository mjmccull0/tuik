package parser

import (
  "github.com/charmbracelet/bubbles/textinput"
	"encoding/json"
	"tuik/components"
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

// ParseComponent is the factory that maps JSON data to Go structs.
func ParseComponent(data map[string]interface{}) components.Component {
	typ, _ := data["type"].(string)

	// Fallback: If 'type' is missing but 'text' is present, treat as a Text component.
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
		
		// Create the library model
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 156
		ti.Width = 20

		base = &components.TextInput{
			ID:          id,
			Placeholder: placeholder,
			Model:       ti, // Pass the initialized model here
		}

	case "list":
		itemsRaw, _ := data["items"].([]interface{})
		var items []components.ListItem
		for _, it := range itemsRaw {
			m, _ := it.(map[string]interface{})
			items = append(items, components.ListItem{
				Text:    m["text"].(string),
				OnPress: m["on-press"].(string),
			})
		}
		base = components.List{Items: items}

	default:
		// Return nil if the type is unknown to avoid crashing the engine.
		return nil
	}

	// Style Wrapping: Check if a nested 'style' object exists.
	if styleData, ok := data["style"].(map[string]interface{}); ok {
		return components.Box{
			Child:  base,
			Styles: extractStyles(styleData),
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
