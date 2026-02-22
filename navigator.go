package main

import (
  // "strings"
  "tuik/components"
)

type Navigator struct {
  Registry   map[string]*components.View
  ActiveView string
}

type NavResult struct {
  NextView string
  Command  string
  IsUpdate bool
}

func (n *Navigator) ProcessAction(action string, data map[string]string) NavResult {
  // 1. Interpolate first
  interpolated := interpolate(action, data)

  // 2. Check if it's a view swap
  if _, exists := n.Registry[interpolated]; exists {
    n.ActiveView = interpolated
    return NavResult{NextView: interpolated, IsUpdate: true}
  }

  // 3. Otherwise, it's a shell command
  return NavResult{Command: interpolated, IsUpdate: false}
}
