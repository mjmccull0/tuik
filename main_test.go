package main

import "testing"

func TestInterpolate(t *testing.T) {
  tests := []struct {
    name     string
    text     string
    data     map[string]string
    expected string
  }{
    {
      name:     "Simple substitution",
      text:     "git commit -m {{.msg}}",
      data:     map[string]string{"msg": "hello"},
      expected: "git commit -m hello",
    },
    {
      name:     "No tags",
      text:     "git status",
      data:     map[string]string{"msg": "hello"},
      expected: "git status",
    },
    {
      name:     "Multiple tags",
      text:     "{{.cmd}} {{.arg}}",
      data:     map[string]string{"cmd": "ls", "arg": "-la"},
      expected: "ls -la",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := interpolate(tt.text, tt.data)
      if result != tt.expected {
        t.Errorf("interpolate() = %v, want %v", result, tt.expected)
      }
    })
  }
}
