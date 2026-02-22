package parser

import (
  "testing"
)

func TestParseConfig(t *testing.T) {
  // Table of test cases
  tests := []struct {
    name    string
    json    string
    wantErr bool
  }{
    {
      name: "Valid simple config",
      json: `{
        "main": "home",
        "views": {
          "home": {
            "id": "home",
            "children": [
              {"type": "text-input", "id": "commit-msg", "placeholder": "Enter msg"}
            ]
          }
        }
      }`,
      wantErr: false,
    },
    {
      name:    "Invalid JSON syntax",
      json:    `{ "main": "home", "views": { "unfinished"... }`,
      wantErr: true,
    },
    {
      name: "Missing main view",
      json: `{ "views": {} }`,
      wantErr: false, // Currently our parser doesn't enforce this, but we could!
    },
    {
      name: "Nested Box config",
      json: `{
          "main": "home",
          "views": {
              "home": {
                  "id": "home",
                  "children": [
                      {
                          "type": "box",
                          "children": [
                              {"type": "text-input", "id": "inner-input"}
                          ]
                      }
                  ]
              }
          }
      }`,
      wantErr: false,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      _, err := ParseConfig([]byte(tt.json))
      if (err != nil) != tt.wantErr {
        t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
      }
    })
  }
}
