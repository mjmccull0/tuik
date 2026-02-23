package components

import "testing"

// mockComponent helps us track if methods were actually called
type mockComponent struct {
	Component // Embed interface to skip implementing everything
	focused   bool
}

func (m *mockComponent) Focus()           { m.focused = true }
func (m *mockComponent) Blur()            { m.focused = false }
func (m *mockComponent) IsFocusable() bool { return true }
func (m *mockComponent) Render(ctx Context) string { return "mock" }

func TestBoxFocusDelegation(t *testing.T) {
    // 1. Create the button pointer
    btn := &Button{ID: "btn-1"} 
    
    // 2. Put it in the box
    box := &Box{
        Children: []Component{btn},
    }

    // 3. Call Focus
    box.Focus()

    // 4. IMPORTANT: Cast the child BACK to a button to check it
    // This ensures we are checking the component actually held by the box
    childFromBox := box.Children[0].(*Button)

    if !childFromBox.focused {
        t.Errorf("Box.Focus() was called, but the child inside the box was not updated")
    }
}
