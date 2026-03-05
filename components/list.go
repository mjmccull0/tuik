package components

import (
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"tuik/utils"
)

// Ensure List implements Component
var _ Component = (*List)(nil)

func (l *List) Render(ctx *Context) string {
	// 1. Resolve the input
	// If input is a string like "{{.types}}", we resolve it via ctx
	items := l.resolveItems(ctx)
	if len(items) == 0 {
		return "  (Loading or No Files Found...)"
	}

	var s strings.Builder
	for i, item := range items {
		cursor := "  "
		if l.Cursor == i {
			cursor = "> "
		}
		// Resolve individual item text in case it's a template
		label := ctx.Resolve(item.Text)
		s.WriteString(cursor + label + "\n")
	}
	return s.String()
}

func (l *List) resolveItems(ctx *Context) []ListItem {
    if items, ok := l.Input.([]ListItem); ok {
        return items
    }

    if cmdStr, ok := l.Input.(string); ok {
        // THIS IS THE MISSING LINK:
        // This will finally trigger your "REPLACEMENT INPUT" logs.
        interpolated := ctx.ReplacePlaceholders(cmdStr)
        
        // If we haven't fetched data yet, we need to return "Loading"
        // and let the Navigator/Main loop handle the actual execution.
        return []ListItem{{Text: "Loading command: " + interpolated}} 
    }

    return []ListItem{}
}

func (l *List) Update(msg tea.Msg, ctx *Context) (Component, tea.Cmd) {
	  switch msg := msg.(type) {
				case ListHydrationMsg:
				// Only update if this message is meant for THIS list
				if msg.ID == l.ID {
					l.Input = msg.Items
					// Reset cursor to the top since the list content changed
					l.Cursor = 0 
					return l, nil
				}
	  }
		// 2. Resolve items for the current render/interaction cycle
    items := l.resolveItems(ctx)
    if len(items) == 0 {
        return l, nil 
    }

    // Ensure the cursor hasn't drifted out of bounds
    if l.Cursor >= len(items) {
        l.Cursor = 0
    }

    // 2. HANDLE INTERACTION
    switch msg := msg.(type) {
				case tea.KeyMsg:
        switch msg.String() {
        case "j", "down":
            if l.Cursor < len(items)-1 { l.Cursor++ }
        case "k", "up":
            if l.Cursor > 0 { l.Cursor-- }
        case "enter":
            // We already have 'items' from above, no need to re-resolve
            if l.Cursor >= 0 && l.Cursor < len(items) {
                selected := items[l.Cursor]

				        action := selected.OnPress
				        if action == "" {
					          action = l.GetAction()
								}
                utils.Log("LIST ENTER: Selected='%s' ID='%s' Action='%s'", selected.Text, l.ID, action)

                if l.ID != "" {
                    // This now works because we defined Set above
                    ctx.Set(l.ID, selected.Text)
					          utils.Log("CONTEXT SET: %s = %s", l.ID, items[l.Cursor].Text)
                }

                return l, func() tea.Msg {
                    return ActionMsg{
                        ID:     l.ID,
                        Action: action,
                    }
                }
            }
        }
    }
    return l, nil
}

func (l *List) Blur() {}
func (l *List) Focus() {}
func (l *List) IsFocusable() bool { return true }
func (l *List) GetAction() string {
	items := l.resolveItems(&Context{})
	if l.Cursor >= 0 && l.Cursor < len(items) {
		// If the specific item has an on-press, use it.
		// Otherwise, use the list's general on-select.
		if items[l.Cursor].OnPress != "" {
			return items[l.Cursor].OnPress
		}
	}
	return l.OnSelect
}
func (l *List) GetID() string     { return l.ID }
func (l *List) GetType()    string { return "list" }
func (l *List) GetValue() string {
    items := l.resolveItems(&Context{})
    if l.Cursor >= 0 && l.Cursor < len(items) {
        // Return the text of the item the user is currently looking at
        return items[l.Cursor].Text
    }
    return ""
}
