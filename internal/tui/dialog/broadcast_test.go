package dialog

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestNewBroadcastSelectsCurrentWorkspace(t *testing.T) {
	model := NewBroadcast([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	}, "ws-2")

	selected := model.selectedWorkspaceIDs()
	if len(selected) != 1 || selected[0] != "ws-2" {
		t.Fatalf("selected = %v, want [ws-2]", selected)
	}
}

func TestNewBroadcastFallsBackToFirstWorkspace(t *testing.T) {
	model := NewBroadcast([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	}, "")

	selected := model.selectedWorkspaceIDs()
	if len(selected) != 1 || selected[0] != "ws-1" {
		t.Fatalf("selected = %v, want [ws-1]", selected)
	}
}

func TestBroadcastUpdateSubmitsSelectedTargets(t *testing.T) {
	model := NewBroadcast([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	}, "ws-1")
	model.prompt.SetValue("ship the feature")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	msg := cmd()
	submit, ok := msg.(BroadcastSubmitMsg)
	if !ok {
		t.Fatalf("message type = %T, want BroadcastSubmitMsg", msg)
	}
	if submit.Prompt != "ship the feature" {
		t.Fatalf("prompt = %q, want %q", submit.Prompt, "ship the feature")
	}
	if len(submit.WorkspaceIDs) != 1 || submit.WorkspaceIDs[0] != "ws-1" {
		t.Fatalf("workspace IDs = %v, want [ws-1]", submit.WorkspaceIDs)
	}
	if updated.focus != broadcastFocusTargets {
		t.Fatalf("focus = %v, want targets", updated.focus)
	}
}

func TestBroadcastUpdateTogglesTargetsAndAllSelection(t *testing.T) {
	model := NewBroadcast([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	}, "ws-1")

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	selected := model.selectedWorkspaceIDs()
	if len(selected) != 2 {
		t.Fatalf("selected after select-all = %v, want both", selected)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	selected = model.selectedWorkspaceIDs()
	if len(selected) != 0 {
		t.Fatalf("selected after clear-all = %v, want none", selected)
	}
}

func TestBroadcastUsesConfiguredKeyMap(t *testing.T) {
	model := NewBroadcast([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	}, "ws-1").WithKeyMap(BroadcastKeyMap{
		Up:    key.NewBinding(key.WithKeys("w")),
		Down:  key.NewBinding(key.WithKeys("s")),
		Tab:   key.NewBinding(key.WithKeys("f")),
		Enter: key.NewBinding(key.WithKeys("e")),
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if updated.cursor != 1 {
		t.Fatalf("cursor after configured down key = %d, want 1", updated.cursor)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
	if updated.focus != broadcastFocusTargets {
		t.Fatalf("focus after default tab = %v, want targets", updated.focus)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if updated.focus != broadcastFocusPrompt {
		t.Fatalf("focus after configured tab key = %v, want prompt", updated.focus)
	}

	view := updated.View()
	if !strings.Contains(view, "f: switch focus") {
		t.Fatalf("view = %q, want configured tab key help", view)
	}
}
