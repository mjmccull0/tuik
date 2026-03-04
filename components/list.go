package components

import (
	"strings"
	tea "github.com/charmbracelet/bubbletea"
)

// Ensure List implements Component
var _ Component = (*List)(nil)

func (l List) Render(ctx Context) string {
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

func (l List) resolveItems(ctx Context) []ListItem {
    if items, ok := l.Input.([]ListItem); ok {
        return items
    }

    if _, ok := l.Input.(string); ok {
        // This gives the user immediate feedback that work is happening
        return []ListItem{{Text: "Loading..."}} 
    }

    return []ListItem{}
}

func (l *List) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
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
            if l.Cursor < len(items)-1 {
                l.Cursor++
            }
        case "k", "up":
            if l.Cursor > 0 {
                l.Cursor--
            }
        case "enter":
					// Now this is safe because we've validated 'items' exists
					selected := items[l.Cursor]

					// SAVE TO CONTEXT: Capture the selection so it's available for the next action
					if l.ID != "" {
						// ctx.Data[l.ID] = selected.OnPress
						ctx.Data[l.ID] = selected.Text
					}

					action := selected.OnPress
					// If the list has a global on-select template, use that instead
					if l.OnSelect != "" {
						action = l.OnSelect
					}

					return l, func() tea.Msg {
						return ActionMsg{
							ID:     l.ID,
							Action: action,
						}
					}
        }
    }

    return l, nil
}

func (l List) Blur() {}
func (l List) Focus() {}
func (l List) IsFocusable() bool { return true }
func (l List) GetAction() string {
	items := l.resolveItems(Context{})
	if l.Cursor >= 0 && l.Cursor < len(items) {
		// If the specific item has an on-press, use it.
		// Otherwise, use the list's general on-select.
		if items[l.Cursor].OnPress != "" {
			return items[l.Cursor].OnPress
		}
	}
	return l.OnSelect
}
func (l List) GetID() string     { return l.ID }
func (l List) GetType()    string { return "list" }
func (l List) GetValue() string  {
	items := l.resolveItems(Context{}) // Simple resolve
	if len(items) > 0 && l.Cursor >= 0 && l.Cursor < len(items) {
		return items[l.Cursor].Text
	}
	return ""
}
