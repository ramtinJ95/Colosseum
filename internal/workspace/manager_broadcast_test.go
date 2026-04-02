package workspace

import (
	"errors"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
)

func TestManagerBroadcast(t *testing.T) {
	h := newManagerHarness(t)
	claude, codex := h.mustCreateBroadcastPair(t)

	result, err := h.mgr.Broadcast(h.ctx, "implement the feature", []string{codex.ID, claude.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if result.Requested != 2 {
		t.Fatalf("requested = %d, want 2", result.Requested)
	}
	if len(result.Delivered) != 2 {
		t.Fatalf("delivered = %d, want 2", len(result.Delivered))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("failed = %d, want 0", len(result.Failed))
	}
	if len(h.sessions.sendKeysCalls) != 2 {
		t.Fatalf("send-keys calls = %d, want 2", len(h.sessions.sendKeysCalls))
	}
	if h.sessions.sendKeysCalls[0].Target != codex.PaneTargets["agent"] {
		t.Fatalf("first target = %q, want %q", h.sessions.sendKeysCalls[0].Target, codex.PaneTargets["agent"])
	}
	if h.sessions.sendKeysCalls[0].Keys != "implement the feature" {
		t.Fatalf("first keys = %q, want broadcast prompt", h.sessions.sendKeysCalls[0].Keys)
	}
	if h.sessions.sendKeysCalls[0].Opts.InputDelay != 0 {
		t.Fatalf("first input delay = %v, want 0 for codex", h.sessions.sendKeysCalls[0].Opts.InputDelay)
	}
	if !h.sessions.sendKeysCalls[0].Opts.ForcePaste {
		t.Fatal("first send should force paste for codex single-line prompts")
	}
	if h.sessions.sendKeysCalls[1].Target != claude.PaneTargets["agent"] {
		t.Fatalf("second target = %q, want %q", h.sessions.sendKeysCalls[1].Target, claude.PaneTargets["agent"])
	}
	if h.sessions.sendKeysCalls[1].Opts.InputDelay != 100*time.Millisecond {
		t.Fatalf("second input delay = %v, want 100ms for claude", h.sessions.sendKeysCalls[1].Opts.InputDelay)
	}
	if h.sessions.sendKeysCalls[1].Opts.ForcePaste {
		t.Fatal("second send should not force paste for claude single-line prompts")
	}
}

func TestManagerBroadcastContinuesOnSendError(t *testing.T) {
	h := newManagerHarness(t)
	claude, codex := h.mustCreateBroadcastPair(t)
	h.sessions.sendKeysErrs = map[string]error{
		claude.PaneTargets["agent"]: errors.New("tmux down"),
	}

	result, err := h.mgr.Broadcast(h.ctx, "ship it", []string{claude.ID, codex.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if len(result.Delivered) != 1 || result.Delivered[0] != codex.Title {
		t.Fatalf("delivered = %v, want [%s]", result.Delivered, codex.Title)
	}
	if len(result.Failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(result.Failed))
	}
	if result.Failed[0].WorkspaceTitle != claude.Title {
		t.Fatalf("failed workspace = %q, want %q", result.Failed[0].WorkspaceTitle, claude.Title)
	}
	if len(h.sessions.sendKeysCalls) != 2 {
		t.Fatalf("send attempts = %d, want 2", len(h.sessions.sendKeysCalls))
	}
}

func TestManagerBroadcastDisablesBracketedPasteForClaudeMultiline(t *testing.T) {
	h := newManagerHarness(t)
	claude, codex := h.mustCreateBroadcastPair(t)

	if _, err := h.mgr.Broadcast(h.ctx, "line1\nline2", []string{claude.ID, codex.ID}); err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if len(h.sessions.sendKeysCalls) != 2 {
		t.Fatalf("send-keys calls = %d, want 2", len(h.sessions.sendKeysCalls))
	}
	if !h.sessions.sendKeysCalls[0].Opts.DisableBracketedPaste {
		t.Fatal("claude multiline should disable bracketed paste")
	}
	if h.sessions.sendKeysCalls[1].Opts.DisableBracketedPaste {
		t.Fatal("codex multiline should keep bracketed paste enabled")
	}
}

func TestManagerBroadcastRejectsInvalidInput(t *testing.T) {
	h := newManagerHarness(t)

	if _, err := h.mgr.Broadcast(h.ctx, "   ", []string{"ws-1"}); err == nil {
		t.Fatal("expected empty prompt error")
	}
	if _, err := h.mgr.Broadcast(h.ctx, "prompt", nil); err == nil {
		t.Fatal("expected empty workspace selection error")
	}
}

func TestManagerBroadcastReportsMissingWorkspace(t *testing.T) {
	h := newManagerHarness(t)

	result, err := h.mgr.Broadcast(h.ctx, "prompt", []string{"missing"})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if result.Requested != 1 {
		t.Fatalf("requested = %d, want 1", result.Requested)
	}
	if len(result.Delivered) != 0 {
		t.Fatalf("delivered = %d, want 0", len(result.Delivered))
	}
	if len(result.Failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(result.Failed))
	}
}

func TestManagerBroadcastSkipsDuplicateWorkspaceIDs(t *testing.T) {
	h := newManagerHarness(t)
	claude, _ := h.mustCreateBroadcastPair(t)

	result, err := h.mgr.Broadcast(h.ctx, "prompt", []string{claude.ID, claude.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if result.Requested != 1 {
		t.Fatalf("requested = %d, want 1 after duplicate IDs are collapsed", result.Requested)
	}
	if len(result.Delivered) != 1 || result.Delivered[0] != claude.Title {
		t.Fatalf("delivered = %v, want [%s]", result.Delivered, claude.Title)
	}
	if len(h.sessions.sendKeysCalls) != 1 {
		t.Fatalf("send-keys calls = %d, want 1", len(h.sessions.sendKeysCalls))
	}
}

func TestManagerBroadcastFailsWithoutAgentPane(t *testing.T) {
	h := newManagerHarness(t)
	ws := h.mustCreate(t, "no-pane", agent.Claude, "/tmp/project", "main", LayoutAgent)

	stored, found, err := h.store.Get(ws.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("workspace not found")
	}
	delete(stored.PaneTargets, "agent")
	if err := h.store.Update(stored); err != nil {
		t.Fatalf("Update: %v", err)
	}
	h.resetSendKeys()

	result, err := h.mgr.Broadcast(h.ctx, "prompt", []string{ws.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if len(result.Delivered) != 0 {
		t.Fatalf("delivered = %d, want 0", len(result.Delivered))
	}
	if len(result.Failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(result.Failed))
	}
	if len(h.sessions.sendKeysCalls) != 0 {
		t.Fatalf("send-keys calls = %d, want 0", len(h.sessions.sendKeysCalls))
	}
}
