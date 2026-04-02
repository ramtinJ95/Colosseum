package dialog

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRenameConfirmReturnsNewTitle(t *testing.T) {
	model := NewRename("ws-1", "old-title")

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	for _, r := range "new-title" {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected confirm command")
	}

	msg := cmd()
	confirm, ok := msg.(RenameConfirmMsg)
	if !ok {
		t.Fatalf("message type = %T, want RenameConfirmMsg", msg)
	}
	if confirm.WorkspaceID != "ws-1" {
		t.Fatalf("WorkspaceID = %q, want %q", confirm.WorkspaceID, "ws-1")
	}
	if confirm.NewTitle != "new-title" {
		t.Fatalf("NewTitle = %q, want %q", confirm.NewTitle, "new-title")
	}
}

func TestRenameCancelReturnsCancelMsg(t *testing.T) {
	model := NewRename("ws-1", "old-title")

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cancel command")
	}

	msg := cmd()
	if _, ok := msg.(RenameCancelMsg); !ok {
		t.Fatalf("message type = %T, want RenameCancelMsg", msg)
	}
}

func TestRenameUsesConfiguredConfirmBinding(t *testing.T) {
	model := NewRename("ws-1", "demo").WithKeyMap(RenameKeyMap{
		Confirm: key.NewBinding(key.WithKeys("x")),
		Cancel:  key.NewBinding(key.WithKeys("esc")),
	})

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd == nil {
		t.Fatal("expected confirm command")
	}

	msg := cmd()
	if _, ok := msg.(RenameConfirmMsg); !ok {
		t.Fatalf("message type = %T, want RenameConfirmMsg", msg)
	}
}

func TestRenamePreFillsCurrentTitle(t *testing.T) {
	model := NewRename("ws-1", "my-workspace")

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected confirm command")
	}

	msg := cmd()
	confirm, ok := msg.(RenameConfirmMsg)
	if !ok {
		t.Fatalf("message type = %T, want RenameConfirmMsg", msg)
	}
	if confirm.NewTitle != "my-workspace" {
		t.Fatalf("NewTitle = %q, want %q (pre-filled current title)", confirm.NewTitle, "my-workspace")
	}
}
