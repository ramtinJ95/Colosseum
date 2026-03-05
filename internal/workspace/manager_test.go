package workspace

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
)

type mockSessionCreator struct {
	createCalls   []mockCreateCall
	killCalls     []string
	splitCalls    []mockSplitCall
	switchCalls   []string
	sendKeysCalls []mockSendKeysCall
	splitCount    int
	createErr     error
	killErr       error
	splitErrAt    int
	sendKeysErr   error
}

type mockCreateCall struct {
	Name     string
	StartDir string
}

type mockSplitCall struct {
	Session    string
	Horizontal bool
	StartDir   string
}

type mockSendKeysCall struct {
	Target string
	Keys   string
}

func (m *mockSessionCreator) CreateSession(_ context.Context, name string, startDir string) (string, error) {
	m.createCalls = append(m.createCalls, mockCreateCall{Name: name, StartDir: startDir})
	if m.createErr != nil {
		return "", m.createErr
	}
	return "%0", nil
}

func (m *mockSessionCreator) KillSession(_ context.Context, name string) error {
	m.killCalls = append(m.killCalls, name)
	return m.killErr
}

func (m *mockSessionCreator) SplitWindow(_ context.Context, session string, horizontal bool, startDir string) (string, error) {
	m.splitCalls = append(m.splitCalls, mockSplitCall{
		Session:    session,
		Horizontal: horizontal,
		StartDir:   startDir,
	})
	m.splitCount++
	if m.splitErrAt > 0 && m.splitCount == m.splitErrAt {
		return "", errors.New("split failed")
	}
	return "%1", nil
}

func (m *mockSessionCreator) SwitchClient(_ context.Context, name string) error {
	m.switchCalls = append(m.switchCalls, name)
	return nil
}

func (m *mockSessionCreator) SendKeys(_ context.Context, target string, keys string) error {
	m.sendKeysCalls = append(m.sendKeysCalls, mockSendKeysCall{Target: target, Keys: keys})
	return m.sendKeysErr
}

func TestManagerCreate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws, err := mgr.Create(context.Background(), "my-workspace", agent.Claude, "/tmp/project", "feature-branch", LayoutAgentShell)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if ws.Title != "my-workspace" {
		t.Errorf("expected title %q, got %q", "my-workspace", ws.Title)
	}
	if ws.SessionName != "colo-my-workspace" {
		t.Errorf("expected session name %q, got %q", "colo-my-workspace", ws.SessionName)
	}
	if ws.AgentType != agent.Claude {
		t.Errorf("expected agent type %q, got %q", agent.Claude, ws.AgentType)
	}
	if ws.ID == "" {
		t.Error("expected non-empty ID")
	}

	if len(mock.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(mock.createCalls))
	}
	if mock.createCalls[0].Name != "my-workspace" {
		t.Errorf("expected create name %q, got %q", "my-workspace", mock.createCalls[0].Name)
	}

	if len(mock.splitCalls) != 1 {
		t.Fatalf("expected 1 split call for agent-shell layout, got %d", len(mock.splitCalls))
	}

	if ws.PaneTargets["agent"] != "%0" {
		t.Errorf("expected agent pane target %%0, got %q", ws.PaneTargets["agent"])
	}

	if len(mock.sendKeysCalls) != 1 {
		t.Fatalf("expected 1 send-keys call to launch agent, got %d", len(mock.sendKeysCalls))
	}
	if mock.sendKeysCalls[0].Target != "%0" {
		t.Errorf("expected send-keys target %%0, got %q", mock.sendKeysCalls[0].Target)
	}
	if mock.sendKeysCalls[0].Keys != "claude" {
		t.Errorf("expected send-keys %q, got %q", "claude", mock.sendKeysCalls[0].Keys)
	}

	stored, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored workspace, got %d", len(stored))
	}
}

func TestManagerDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws, err := mgr.Create(context.Background(), "to-delete", agent.Codex, "/tmp/project", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mgr.Delete(context.Background(), ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if len(mock.killCalls) != 1 {
		t.Fatalf("expected 1 kill call, got %d", len(mock.killCalls))
	}
	if mock.killCalls[0] != "to-delete" {
		t.Errorf("expected kill name %q, got %q", "to-delete", mock.killCalls[0])
	}

	remaining, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 workspaces after delete, got %d", len(remaining))
	}
}

func TestManagerCreateRollsBackOnSplitError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{splitErrAt: 1}
	mgr := NewManager(store, mock, "colo-")

	if _, err := mgr.Create(context.Background(), "broken", agent.Claude, "/tmp/project", "main", LayoutAgentShell); err == nil {
		t.Fatal("expected create to fail")
	}

	if len(mock.killCalls) != 1 || mock.killCalls[0] != "broken" {
		t.Fatalf("expected rollback kill for broken workspace, got %v", mock.killCalls)
	}

	stored, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("expected no persisted workspace after rollback, got %d", len(stored))
	}
}

func TestManagerCreateReturnsLaunchError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{sendKeysErr: errors.New("send failed")}
	mgr := NewManager(store, mock, "colo-")

	if _, err := mgr.Create(context.Background(), "launch-fail", agent.Claude, "/tmp/project", "main", LayoutAgent); err == nil {
		t.Fatal("expected launch error")
	}

	if len(mock.killCalls) != 1 || mock.killCalls[0] != "launch-fail" {
		t.Fatalf("expected rollback kill for launch-fail workspace, got %v", mock.killCalls)
	}
}

func TestManagerDeleteIgnoresSessionNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws, err := mgr.Create(context.Background(), "already-gone", agent.Codex, "/tmp/project", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mock.killErr = &tmux.TmuxError{Args: []string{"kill-session"}, Stderr: "session not found"}
	if err := mgr.Delete(context.Background(), ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestManagerDeleteReturnsKillError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws, err := mgr.Create(context.Background(), "kill-fail", agent.Codex, "/tmp/project", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mock.killErr = errors.New("tmux unavailable")
	if err := mgr.Delete(context.Background(), ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}

	remaining, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected workspace to remain after failed delete, got %d", len(remaining))
	}
}

func TestManagerList(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ctx := context.Background()
	if _, err := mgr.Create(ctx, "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent); err != nil {
		t.Fatalf("Create ws-1: %v", err)
	}
	if _, err := mgr.Create(ctx, "ws-2", agent.Codex, "/tmp/p2", "dev", LayoutAgentShell); err != nil {
		t.Fatalf("Create ws-2: %v", err)
	}
	if _, err := mgr.Create(ctx, "ws-3", agent.Claude, "/tmp/p3", "feature", LayoutAgentShellLogs); err != nil {
		t.Fatalf("Create ws-3: %v", err)
	}

	workspaces, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(workspaces) != 3 {
		t.Errorf("expected 3 workspaces, got %d", len(workspaces))
	}
}
