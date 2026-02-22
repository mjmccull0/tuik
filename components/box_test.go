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
func (m *mockComponent) Render(ctx RenderContext) string { return "mock" }

func TestBoxFocusDelegation(t *testing.T) {
	mock := &mockComponent{}
	box := Box{
		Child: mock,
	}

	// Test Focus
	box.Focus()
	if !mock.focused {
		t.Error("Box.Focus() was called, but child did not receive it")
	}

	// Test Blur
	box.Blur()
	if mock.focused {
		t.Error("Box.Blur() was called, but child is still focused")
	}
}
