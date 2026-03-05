package main

import (
	"regexp"
	"strings"
)

type Context struct {
	Data   map[string]string // Captured outputs (e.g., last_output)
	Styles map[string]string // Global theme colors/values
}

// Resolve replaces {{.key}} or {{.styles.key}} placeholders with real values.
func (c *Context) Resolve(input string) string {
	if !strings.Contains(input, "{{.") {
		return input
	}

	// Handles {{.key}} and {{.key:-fallback}}
	re := regexp.MustCompile(`\{\{\.([a-zA-Z0-9_\.]+)(?::-(.*?))?\}\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		key := submatches[1]
		fallback := submatches[2]

		// 1. Check for Style resolution: {{.styles.primary}}
		if strings.HasPrefix(key, "styles.") {
			styleKey := strings.TrimPrefix(key, "styles.")
			if val, ok := c.Styles[styleKey]; ok {
				return val
			}
		}

		// 2. Check for Data resolution: {{.last_output}}
		if val, ok := c.Data[key]; ok && val != "" {
			return val
		}

		return fallback
	})
}
