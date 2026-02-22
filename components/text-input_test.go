package components

import (
	"testing"
	"github.com/charmbracelet/bubbles/textinput"
)

func TestTextInputUpdateSync(t *testing.T) {
	ti := &TextInput{
		Model: textinput.New(),
	}
	
	// Simulate the library state changing (as if a user typed "hi")
	ti.Model.SetValue("hi")
	
	// We pass nil msg because we're testing the manual sync in our Update
	ti.Update(nil)

	if ti.Content != "hi" {
		t.Errorf("Expected Content to be 'hi' after sync, got '%s'", ti.Content)
	}
}
