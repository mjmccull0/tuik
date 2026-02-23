package parser

import (
	"encoding/json"
	"tuik/components"

	"github.com/charmbracelet/bubbles/textinput"
)


func parseStaticItems(raw []interface{}) []components.ListItem {
    var items []components.ListItem
    for _, it := range raw {
        if m, ok := it.(map[string]interface{}); ok {
            txt, _ := m["text"].(string)
            act, _ := m["on-press"].(string)
            items = append(items, components.ListItem{Text: txt, OnPress: act})
        }
    }
    return items
}

func ParseConfig(data []byte) (components.Config, error) {
	var raw struct {
		Main  string `json:"main"`
		Views map[string]struct {
			Context  map[string]string      `json:"context"`
			Style    map[string]interface{} `json:"style"`
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
		v := &components.View{
			ID:      viewID,
			Context: make(map[string]any),
		}

		for k, val := range viewData.Context {
			v.Context[k] = val
		}

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
		base = &components.Text{Content: content}

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
    
    var inputData interface{}

    // 1. Check for the new "input" key first
    if rawInput, ok := data["input"]; ok {
        switch v := rawInput.(type) {
        case string:
            // It's a command! Store the string for the Navigator to hydrate later
            inputData = v
        case []interface{}:
            // It's a static list! Convert it to []ListItem
            inputData = parseStaticItems(v)
        }
    } else if itemsRaw, ok := data["items"].([]interface{}); ok {
        // 2. Fallback for your existing "items" key
        inputData = parseStaticItems(itemsRaw)
    }

    base = &components.List{
        ID:       id, 
        Input:    inputData, 
        OnSelect: onSelect,
    }

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
		// Boxes can handle their own styles now
		box := &components.Box{
			ID:       id,
			Children: children,
			Vertical: vertical,
		}
		if styleData, ok := data["style"].(map[string]interface{}); ok {
			box.Styles = extractStyles(styleData)
		}
		base = box

	case "button":
		text, _ := data["text"].(string)
		action, _ := data["on-select"].(string)
		if action == "" {
			action, _ = data["on-press"].(string)
		}
		id, _ := data["id"].(string)
		btn := &components.Button{
			ID:     id,
			Text:   text,
			Action: action,
		}
		// Attach style directly to button instead of creating a wrapper box
		if styleData, ok := data["style"].(map[string]interface{}); ok {
			btn.Styles = extractStyles(styleData)
		}
		base = btn

	default:
		return nil
	}

	return base
}

func extractStyles(s map[string]interface{}) components.StyleConfig {
	styles := components.StyleConfig{}

	if val, ok := s["padding"].(float64); ok {
		// Map simple padding to all sides
		p := int(val)
		styles.PaddingTop, styles.PaddingRight, styles.PaddingBottom, styles.PaddingLeft = p, p, p, p
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
