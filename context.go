package main

import (
  "regexp"
  "strings"
)

type Context struct {
  Data   map[string]string
  Styles map[string]string
}

func (c *Context) Resolve(input string) string {
  if !strings.Contains(input, "{{.") {
    return input
  }

  re := regexp.MustCompile(`\{\{\.([a-zA-Z0-9_\.]+)(?::-(.*?))?\}\}`)

  return re.ReplaceAllStringFunc(input, func(match string) string {
    submatches := re.FindStringSubmatch(match)
    key := submatches[1]
    fallback := submatches[2]

    // Handle {{.styles.primary}}
    if strings.HasPrefix(key, "styles.") {
      styleKey := strings.TrimPrefix(key, "styles.")
      if val, ok := c.Styles[styleKey]; ok {
        return val
      }
    }

    // Handle {{.last_output}} or other data
    if val, ok := c.Data[key]; ok && val != "" {
      return val
    }

    return fallback
  })
}
