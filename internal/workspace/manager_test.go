package workspace

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
)

type mockSessionCreator struct {
	createCalls   []mockCreateCall
	killCalls     []string
	splitCalls    []mockSplitCall
	switchCalls   []string
	sendKeysCalls []mockSendKeysCall
	createCount   int
	splitCount    int
	createErr     error
	killErr       error
	splitErrAt    int
	sendKeysErr   error
	sendKeysErrs  map[string]error
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
	Opts   tmux.SendOptions
}

func (m *mockSessionCreator) CreateSession(_ context.Context, name string, startDir string) (string, error) {
	m.createCalls = append(m.createCalls, mockCreateCall{Name: name, StartDir: startDir})
	if m.createErr != nil {
		return "", m.createErr
	}
	paneID := fmt.Sprintf("%%%d", m.createCount)
	m.createCount++
	return paneID, nil
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

func (m *mockSessionCreator) SendKeys(_ context.Context, target string, keys string, opts tmux.SendOptions) error {
	m.sendKeysCalls = append(m.sendKeysCalls, mockSendKeysCall{Target: target, Keys: keys, Opts: opts})
	if err, ok := m.sendKeysErrs[target]; ok {
		return err
	}
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
	if mock.createCalls[0].Name != "colo-my-workspace" {
		t.Errorf("expected create name %q, got %q", "colo-my-workspace", mock.createCalls[0].Name)
	}

	if len(mock.splitCalls) != 1 {
		t.Fatalf("expected 1 split call for agent-shell layout, got %d", len(mock.splitCalls))
	}
	if mock.splitCalls[0].Session != "colo-my-workspace" {
		t.Errorf("expected split session %q, got %q", "colo-my-workspace", mock.splitCalls[0].Session)
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
	if mock.killCalls[0] != "colo-to-delete" {
		t.Errorf("expected kill name %q, got %q", "colo-to-delete", mock.killCalls[0])
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

	if len(mock.killCalls) != 1 || mock.killCalls[0] != "colo-broken" {
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

	if len(mock.killCalls) != 1 || mock.killCalls[0] != "colo-launch-fail" {
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

func TestManagerSwitchToUsesStoredSessionName(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws, err := mgr.Create(context.Background(), "switch-me", agent.Claude, "/tmp/project", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mgr.SwitchTo(context.Background(), ws.ID); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	if len(mock.switchCalls) != 1 {
		t.Fatalf("expected 1 switch call, got %d", len(mock.switchCalls))
	}
	if mock.switchCalls[0] != ws.SessionName {
		t.Fatalf("switch target = %q, want %q", mock.switchCalls[0], ws.SessionName)
	}
}

func TestManagerSwitchToFallsBackToPrefixedTitle(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
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

	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")
	if err := mgr.SwitchTo(context.Background(), ws.ID); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	if len(mock.switchCalls) != 1 {
		t.Fatalf("expected 1 switch call, got %d", len(mock.switchCalls))
	}
	if mock.switchCalls[0] != "colo-legacy" {
		t.Fatalf("switch target = %q, want %q", mock.switchCalls[0], "colo-legacy")
	}
}

func TestManagerBroadcast(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws1, err := mgr.Create(context.Background(), "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-1: %v", err)
	}
	ws2, err := mgr.Create(context.Background(), "ws-2", agent.Codex, "/tmp/p2", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-2: %v", err)
	}

	mock.sendKeysCalls = nil

	result, err := mgr.Broadcast(context.Background(), "implement the feature", []string{ws2.ID, ws1.ID})
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
	if len(mock.sendKeysCalls) != 2 {
		t.Fatalf("expected 2 send-keys calls, got %d", len(mock.sendKeysCalls))
	}
	if mock.sendKeysCalls[0].Target != ws2.PaneTargets["agent"] {
		t.Fatalf("first send target = %q, want %q", mock.sendKeysCalls[0].Target, ws2.PaneTargets["agent"])
	}
	if mock.sendKeysCalls[0].Keys != "implement the feature" {
		t.Fatalf("first send keys = %q, want broadcast prompt", mock.sendKeysCalls[0].Keys)
	}
	if mock.sendKeysCalls[0].Opts.InputDelay != 0 {
		t.Fatalf("first send input delay = %v, want 0 for codex", mock.sendKeysCalls[0].Opts.InputDelay)
	}
	if !mock.sendKeysCalls[0].Opts.ForcePaste {
		t.Fatal("first send should force paste for codex single-line prompts")
	}
	if mock.sendKeysCalls[1].Target != ws1.PaneTargets["agent"] {
		t.Fatalf("second send target = %q, want %q", mock.sendKeysCalls[1].Target, ws1.PaneTargets["agent"])
	}
	if mock.sendKeysCalls[1].Opts.InputDelay != 100*time.Millisecond {
		t.Fatalf("second send input delay = %v, want 100ms for claude", mock.sendKeysCalls[1].Opts.InputDelay)
	}
	if mock.sendKeysCalls[1].Opts.ForcePaste {
		t.Fatal("second send should not force paste for claude single-line prompts")
	}
}

func TestManagerBroadcastContinuesOnSendError(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws1, err := mgr.Create(context.Background(), "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-1: %v", err)
	}
	ws2, err := mgr.Create(context.Background(), "ws-2", agent.Codex, "/tmp/p2", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-2: %v", err)
	}

	mock.sendKeysCalls = nil
	mock.sendKeysErrs = map[string]error{
		ws1.PaneTargets["agent"]: errors.New("tmux down"),
	}

	result, err := mgr.Broadcast(context.Background(), "ship it", []string{ws1.ID, ws2.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if len(result.Delivered) != 1 || result.Delivered[0] != "ws-2" {
		t.Fatalf("delivered = %v, want [ws-2]", result.Delivered)
	}
	if len(result.Failed) != 1 {
		t.Fatalf("failed = %d, want 1", len(result.Failed))
	}
	if result.Failed[0].WorkspaceTitle != "ws-1" {
		t.Fatalf("failed workspace = %q, want ws-1", result.Failed[0].WorkspaceTitle)
	}
	if len(mock.sendKeysCalls) != 2 {
		t.Fatalf("expected 2 send attempts, got %d", len(mock.sendKeysCalls))
	}
}

func TestManagerBroadcastDisablesBracketedPasteForClaudeMultiline(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ws1, err := mgr.Create(context.Background(), "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-1: %v", err)
	}
	ws2, err := mgr.Create(context.Background(), "ws-2", agent.Codex, "/tmp/p2", "main", LayoutAgent)
	if err != nil {
		t.Fatalf("Create ws-2: %v", err)
	}

	mock.sendKeysCalls = nil

	_, err = mgr.Broadcast(context.Background(), "line1\nline2", []string{ws1.ID, ws2.ID})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}

	if len(mock.sendKeysCalls) != 2 {
		t.Fatalf("expected 2 send-keys calls, got %d", len(mock.sendKeysCalls))
	}
	if !mock.sendKeysCalls[0].Opts.DisableBracketedPaste {
		t.Fatal("claude multiline should disable bracketed paste")
	}
	if mock.sendKeysCalls[1].Opts.DisableBracketedPaste {
		t.Fatal("codex multiline should keep bracketed paste enabled")
	}
}

func TestManagerBroadcastRejectsInvalidInput(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	if _, err := mgr.Broadcast(context.Background(), "   ", []string{"ws-1"}); err == nil {
		t.Fatal("expected empty prompt error")
	}
	if _, err := mgr.Broadcast(context.Background(), "prompt", nil); err == nil {
		t.Fatal("expected empty workspace selection error")
	}
}

func TestManagerBroadcastReportsMissingWorkspace(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	result, err := mgr.Broadcast(context.Background(), "prompt", []string{"missing"})
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
