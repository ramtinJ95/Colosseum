package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
)

func TestManagerCreate(t *testing.T) {
	h := newManagerHarness(t)

	ws := h.mustCreate(t, "my-workspace", agent.Claude, "/tmp/project", "feature-branch", LayoutAgentShell)

	if ws.Title != "my-workspace" {
		t.Errorf("title = %q, want %q", ws.Title, "my-workspace")
	}
	if ws.SessionName != "colo-my-workspace" {
		t.Errorf("session name = %q, want %q", ws.SessionName, "colo-my-workspace")
	}
	if ws.AgentType != agent.Claude {
		t.Errorf("agent type = %q, want %q", ws.AgentType, agent.Claude)
	}
	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}

	if len(h.sessions.createCalls) != 1 {
		t.Fatalf("create calls = %d, want 1", len(h.sessions.createCalls))
	}
	if h.sessions.createCalls[0].Name != "colo-my-workspace" {
		t.Errorf("create name = %q, want %q", h.sessions.createCalls[0].Name, "colo-my-workspace")
	}

	if len(h.sessions.splitCalls) != 1 {
		t.Fatalf("split calls = %d, want 1 for agent-shell layout", len(h.sessions.splitCalls))
	}
	if h.sessions.splitCalls[0].Session != "colo-my-workspace" {
		t.Errorf("split session = %q, want %q", h.sessions.splitCalls[0].Session, "colo-my-workspace")
	}

	if ws.PaneTargets["agent"] != "%0" {
		t.Errorf("agent pane = %q, want %%0", ws.PaneTargets["agent"])
	}

	if len(h.sessions.sendKeysCalls) != 1 {
		t.Fatalf("send-keys calls = %d, want 1", len(h.sessions.sendKeysCalls))
	}
	if h.sessions.sendKeysCalls[0].Target != "%0" {
		t.Errorf("send-keys target = %q, want %%0", h.sessions.sendKeysCalls[0].Target)
	}
	launchKeys := h.sessions.sendKeysCalls[0].Keys
	if !strings.Contains(launchKeys, "COLOSSEUM_ENV='1'") || !strings.Contains(launchKeys, "COLOSSEUM_WORKSPACE_ID='") || !strings.Contains(launchKeys, "COLOSSEUM_PANE='agent'") || !strings.HasSuffix(launchKeys, " claude") {
		t.Errorf("send-keys = %q, want Colosseum env vars and claude launch", launchKeys)
	}

	if got := len(h.mustList(t)); got != 1 {
		t.Fatalf("stored workspaces = %d, want 1", got)
	}
}

func TestManagerCreateRejectsDashboardTitle(t *testing.T) {
	h := newManagerHarness(t)

	if _, err := h.mgr.Create(h.ctx, "dashboard", agent.Claude, "/tmp/project", "main", LayoutAgent); err == nil {
		t.Fatal("expected reserved dashboard title error")
	}
	if len(h.sessions.createCalls) != 0 {
		t.Fatalf("create calls = %d, want 0", len(h.sessions.createCalls))
	}
}

func TestManagerDelete(t *testing.T) {
	h := newManagerHarness(t)
	ws := h.mustCreate(t, "to-delete", agent.Codex, "/tmp/project", "main", LayoutAgent)

	if err := h.mgr.Delete(h.ctx, ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if len(h.sessions.killCalls) != 1 {
		t.Fatalf("kill calls = %d, want 1", len(h.sessions.killCalls))
	}
	if h.sessions.killCalls[0] != "colo-to-delete" {
		t.Errorf("kill session = %q, want %q", h.sessions.killCalls[0], "colo-to-delete")
	}
	if got := len(h.mustList(t)); got != 0 {
		t.Fatalf("remaining workspaces = %d, want 0", got)
	}
}

func TestManagerCreateRollsBackOnSplitError(t *testing.T) {
	h := newManagerHarness(t)
	h.sessions.splitErrAt = 1

	if _, err := h.mgr.Create(h.ctx, "broken", agent.Claude, "/tmp/project", "main", LayoutAgentShell); err == nil {
		t.Fatal("expected create to fail")
	}

	if len(h.sessions.killCalls) != 1 || h.sessions.killCalls[0] != "colo-broken" {
		t.Fatalf("kill calls = %v, want rollback for broken workspace", h.sessions.killCalls)
	}
	if got := len(h.mustList(t)); got != 0 {
		t.Fatalf("stored workspaces after rollback = %d, want 0", got)
	}
}

func TestManagerCreateReturnsLaunchError(t *testing.T) {
	h := newManagerHarness(t)
	h.sessions.sendKeysErr = errors.New("send failed")

	if _, err := h.mgr.Create(h.ctx, "launch-fail", agent.Claude, "/tmp/project", "main", LayoutAgent); err == nil {
		t.Fatal("expected launch error")
	}
	if len(h.sessions.killCalls) != 1 || h.sessions.killCalls[0] != "colo-launch-fail" {
		t.Fatalf("kill calls = %v, want rollback for launch-fail workspace", h.sessions.killCalls)
	}
}

func TestManagerDeleteIgnoresSessionNotFound(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
	}{
		{name: "legacy message", stderr: "session not found"},
		{name: "cant find session", stderr: "can't find session: colo-already-gone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newManagerHarness(t)
			ws := h.mustCreate(t, "already-gone", agent.Codex, "/tmp/project", "main", LayoutAgent)

			h.sessions.killErr = &tmux.TmuxError{Args: []string{"kill-session"}, Stderr: tt.stderr}
			if err := h.mgr.Delete(h.ctx, ws.ID); err != nil {
				t.Fatalf("Delete: %v", err)
			}

			if got := len(h.mustList(t)); got != 0 {
				t.Fatalf("remaining workspaces = %d, want 0", got)
			}
		})
	}
}

func TestManagerDeleteReturnsKillError(t *testing.T) {
	h := newManagerHarness(t)
	ws := h.mustCreate(t, "kill-fail", agent.Codex, "/tmp/project", "main", LayoutAgent)

	h.sessions.killErr = errors.New("tmux unavailable")
	if err := h.mgr.Delete(h.ctx, ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if got := len(h.mustList(t)); got != 1 {
		t.Fatalf("remaining workspaces = %d, want 1", got)
	}
}

func TestManagerList(t *testing.T) {
	h := newManagerHarness(t)

	h.mustCreate(t, "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent)
	h.mustCreate(t, "ws-2", agent.Codex, "/tmp/p2", "dev", LayoutAgentShell)
	h.mustCreate(t, "ws-3", agent.Claude, "/tmp/p3", "feature", LayoutAgentShellLogs)

	if got := len(h.mustList(t)); got != 3 {
		t.Errorf("workspaces = %d, want 3", got)
	}
}

func TestManagerSwitchToUsesStoredSessionName(t *testing.T) {
	h := newManagerHarness(t)
	ws := h.mustCreate(t, "switch-me", agent.Claude, "/tmp/project", "main", LayoutAgent)

	if err := h.mgr.SwitchTo(h.ctx, ws.ID); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}
	if len(h.sessions.switchCalls) != 1 {
		t.Fatalf("switch calls = %d, want 1", len(h.sessions.switchCalls))
	}
	if h.sessions.switchCalls[0] != ws.SessionName {
		t.Fatalf("switch target = %q, want %q", h.sessions.switchCalls[0], ws.SessionName)
	}
}

func TestManagerSwitchToFallsBackToPrefixedTitle(t *testing.T) {
	store := newTestStore(t)
	ws := Workspace{
		ID:          "ws-1",
		Title:       "legacy",
		AgentType:   agent.Claude,
		ProjectPath: "/tmp/project",
		Layout:      LayoutAgent,
		Status:      agent.StatusIdle,
		PaneTargets: map[string]string{"agent": "%0"},
	}
	if err := store.Add(ws); err != nil {
		t.Fatalf("Add: %v", err)
	}

	h := newManagerHarnessWithStore(t, store)
	if err := h.mgr.SwitchTo(h.ctx, ws.ID); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}
	if len(h.sessions.switchCalls) != 1 {
		t.Fatalf("switch calls = %d, want 1", len(h.sessions.switchCalls))
	}
	if h.sessions.switchCalls[0] != "colo-legacy" {
		t.Fatalf("switch target = %q, want %q", h.sessions.switchCalls[0], "colo-legacy")
	}
}

func TestManagerRename(t *testing.T) {
	h := newManagerHarness(t)
	ws := h.mustCreate(t, "original", agent.Claude, "/tmp/project", "main", LayoutAgent)

	if err := h.mgr.Rename(h.ctx, ws.ID, "renamed"); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	if len(h.sessions.renameCalls) != 1 {
		t.Fatalf("rename calls = %d, want 1", len(h.sessions.renameCalls))
	}
	if h.sessions.renameCalls[0].OldName != "colo-original" {
		t.Errorf("old session name = %q, want %q", h.sessions.renameCalls[0].OldName, "colo-original")
	}
	if h.sessions.renameCalls[0].NewName != "colo-renamed" {
		t.Errorf("new session name = %q, want %q", h.sessions.renameCalls[0].NewName, "colo-renamed")
	}

	updated, found, err := h.store.Get(ws.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("workspace not found after rename")
	}
	if updated.Title != "renamed" {
		t.Errorf("title = %q, want %q", updated.Title, "renamed")
	}
	if updated.SessionName != "colo-renamed" {
		t.Errorf("session name = %q, want %q", updated.SessionName, "colo-renamed")
	}
}

func TestManagerRenameRollsBackSessionWhenStatePersistFails(t *testing.T) {
	store, path := newTestStoreWithPath(t)
	h := newManagerHarnessWithStore(t, store)
	ws := h.mustCreate(t, "original", agent.Claude, "/tmp/project", "main", LayoutAgent)

	stateDir := filepath.Dir(path)
	if err := os.Chmod(stateDir, 0o555); err != nil {
		t.Fatalf("Chmod(%q): %v", stateDir, err)
	}
	defer func() {
		_ = os.Chmod(stateDir, 0o755)
	}()

	if err := h.mgr.Rename(h.ctx, ws.ID, "renamed"); err == nil {
		t.Fatal("expected rename to fail when workspace state cannot be persisted")
	}

	if len(h.sessions.renameCalls) != 2 {
		t.Fatalf("rename calls = %d, want 2", len(h.sessions.renameCalls))
	}
	if h.sessions.renameCalls[0].OldName != "colo-original" || h.sessions.renameCalls[0].NewName != "colo-renamed" {
		t.Fatalf("initial rename = %+v, want old=%q new=%q", h.sessions.renameCalls[0], "colo-original", "colo-renamed")
	}
	if h.sessions.renameCalls[1].OldName != "colo-renamed" || h.sessions.renameCalls[1].NewName != "colo-original" {
		t.Fatalf("rollback rename = %+v, want old=%q new=%q", h.sessions.renameCalls[1], "colo-renamed", "colo-original")
	}

	stored, found, err := h.store.Get(ws.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("workspace not found after failed rename")
	}
	if stored.Title != "original" {
		t.Errorf("title = %q, want %q", stored.Title, "original")
	}
	if stored.SessionName != "colo-original" {
		t.Errorf("session name = %q, want %q", stored.SessionName, "colo-original")
	}
}

func TestManagerRenameRejectsInvalidTitles(t *testing.T) {
	tests := []struct {
		name     string
		newTitle string
		setup    func(t *testing.T, h *managerHarness, ws *Workspace)
	}{
		{
			name:     "empty title",
			newTitle: "  ",
		},
		{
			name:     "duplicate title",
			newTitle: "first",
			setup: func(t *testing.T, h *managerHarness, _ *Workspace) {
				t.Helper()
				h.mustCreate(t, "first", agent.Codex, "/tmp/project", "main", LayoutAgent)
			},
		},
		{
			name:     "dashboard title",
			newTitle: "dashboard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newManagerHarness(t)
			ws := h.mustCreate(t, "original", agent.Claude, "/tmp/project", "main", LayoutAgent)
			if tt.setup != nil {
				tt.setup(t, h, ws)
			}

			if err := h.mgr.Rename(h.ctx, ws.ID, tt.newTitle); err == nil {
				t.Fatalf("expected rename to fail for %q", tt.newTitle)
			}
		})
	}
}

func TestManagerRenameNotFound(t *testing.T) {
	h := newManagerHarness(t)

	if err := h.mgr.Rename(h.ctx, "nonexistent", "new-title"); err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
}
