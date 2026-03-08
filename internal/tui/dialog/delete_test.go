package dialog

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDeleteUsesConfiguredConfirmBinding(t *testing.T) {
	model := NewDelete("ws-1", "demo", false).WithKeyMap(DeleteKeyMap{
		Confirm: key.NewBinding(key.WithKeys("x")),
		Cancel:  key.NewBinding(key.WithKeys("esc")),
	})

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected confirm command")
	}

	msg := cmd()
	if _, ok := msg.(DeleteConfirmMsg); !ok {
		t.Fatalf("message type = %T, want DeleteConfirmMsg", msg)
	}
}
