package main

import (
	"testing"
	"tuik/components"
)


func TestNavigatorFlow(t *testing.T) {
  reg := map[string]*components.View{
    "home":   {ID: "home"},
    "commit": {ID: "commit"},
  }
  nav := &Navigator{Registry: reg, ActiveView: "home"}

  // Test 1: Navigation Swap
  res := nav.ProcessAction("commit", nil)
  if res.NextView != "commit" {
    t.Errorf("Expected to move to 'commit' view, got %s", res.NextView)
  }

  // Test 2: Interpolated Command
  data := map[string]string{"msg": "fix-bugs"}
  res = nav.ProcessAction("git commit -m {{.msg}}", data)
  if res.Command != "git commit -m fix-bugs" {
    t.Errorf("Interpolation failed, got: %s", res.Command)
  }
}

func TestNavigatorViewSwapState(t *testing.T) {
	reg := map[string]*components.View{
		"home": {ID: "home"},
		"next": {ID: "next"},
	}
	nav := &Navigator{Registry: reg, ActiveView: "home"}

	res := nav.ProcessAction("next", nil)

	if nav.ActiveView != "next" {
		t.Errorf("Navigator internal state didn't update. Expected 'next', got %s", nav.ActiveView)
	}
    
    if !res.IsUpdate {
        t.Errorf("Expected IsUpdate to be true for a view swap")
    }
}
