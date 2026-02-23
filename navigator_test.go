package main

import (
	"testing"
	"tuik/components"
)


func TestNavigatorFlow(t *testing.T) {
    // 1. Update map to use pointers if that's what Navigator expects
    reg := map[string]components.View{
        "home":   {ID: "home"},
        "commit": {ID: "commit"},
    }

    nav := Navigator{
        Views:        reg,
        ActiveViewID: "home",
        Context:      components.Context{Data: make(map[string]string)}, // Ensure Context is init
    }

    // Test 1: Navigation Swap
    // Note: If ProcessAction returns ActionResult, check the field names
    res := nav.ProcessAction("commit", nil)
    if nav.ActiveViewID != "commit" {
        t.Errorf("Expected ActiveViewID to be 'commit', got %s", nav.ActiveViewID)
    }

    // Test 2: Interpolated Command
    data := map[string]string{"msg": "fix-bugs"}
    res = nav.ProcessAction("git commit -m {{.msg}}", data)
    if res.Command != "git commit -m fix-bugs" {
        t.Errorf("Interpolation failed, got: %s", res.Command)
    }
}

func TestNavigatorViewSwapState(t *testing.T) {
	reg := map[string]components.View{
		"home": {ID: "home"},
		"next": {ID: "next"},
	}
	nav := Navigator{
		Views:   reg,
		ActiveViewID: "main",
	}

	res := nav.ProcessAction("next", nil)

	if nav.ActiveViewID != "next" {
		t.Errorf("Navigator internal state didn't update. Expected 'next', got %s", nav.ActiveViewID)
	}
    
    if !res.IsUpdate {
        t.Errorf("Expected IsUpdate to be true for a view swap")
    }
}
