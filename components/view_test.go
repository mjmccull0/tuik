package components

import "testing"

func TestViewInitialization(t *testing.T) {
  v := &View{
    ID: "main-view",
    Children: []Component{
      &TextInput{ID: "input-1"},
      &TextInput{ID: "input-2"},
    },
  }

  if len(v.Children) != 2 {
    t.Errorf("Expected 2 children, got %d", len(v.Children))
  }

  if v.ID != "main-view" {
    t.Errorf("Expected ID 'main-view', got %s", v.ID)
  }
}
