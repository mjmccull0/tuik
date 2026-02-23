package components

import (
	"strings"
	"os/exec"
	tea "github.com/charmbracelet/bubbletea"
	"tuik/utils"
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
	// Case 1: Dynamic Shell Command
	if cmdStr, ok := l.Input.(string); ok {
		utils.Log("Executing shell command: %s", cmdStr)

		out, err := exec.Command("sh", "-c", cmdStr).CombinedOutput()
		if err != nil {
			utils.Log("SHELL ERROR: %v | Output: %s", err, string(out))
			return []ListItem{{Text: "Error: " + err.Error()}}
		}

		outputStr := string(out)
		// utils.Log("SHELL SUCCESS: Output: %s", outputStr)

		// --- THE MISSING LOGIC START ---
		lines := strings.Split(strings.TrimSpace(outputStr), "\n")
		var items []ListItem
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				items = append(items, ListItem{Text: strings.TrimSpace(line)})
			}
		}
		return items
		// --- THE MISSING LOGIC END ---
	}

	// Case 2: Static ListItems (already parsed from JSON)
	if items, ok := l.Input.([]ListItem); ok {
		return items
	}

	// Case 3: Simple string slice
	if strs, ok := l.Input.([]string); ok {
		items := make([]ListItem, len(strs))
		for i, s := range strs {
			items[i] = ListItem{Text: s}
		}
		return items
	}

	return []ListItem{}
}

func (l *List) Update(msg tea.Msg, ctx Context) (Component, tea.Cmd) {
    // 1. UNPACK AND VALIDATE (The code goes here)
    items, ok := l.Input.([]ListItem)
    if !ok || len(items) == 0 {
        return l, nil // Stop early if there's no data to act on
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
