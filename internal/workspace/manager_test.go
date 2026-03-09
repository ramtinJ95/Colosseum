package workspace

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/worktrunk"
)

type mockSessionCreator struct {
	createCalls   []mockCreateCall
	killCalls     []string
	splitCalls    []mockSplitCall
	switchCalls   []string
	sendKeysCalls []mockSendKeysCall
	events        *[]string
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

type mockCheckoutLifecycle struct {
	available        bool
	resolveSnapshots map[string]worktrunk.Snapshot
	resolveCalls     []string
	createSnapshots  map[string]worktrunk.Snapshot
	createCalls      []mockWorktreeCreateCall
	removeCalls      []mockWorktreeRemoveCall
	events           *[]string
	createErr        error
	removeErr        error
}

type mockGitInspector struct {
	repoRoots       map[string]string
	currentBranches map[string]string
	defaultBranches map[string]string
	mergeBases      map[string]string
}

type mockWorktreeCreateCall struct {
	RepoPath string
	Branch   string
	Base     string
}

type mockWorktreeRemoveCall struct {
	RepoPath string
	Branches []string
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
	if m.events != nil {
		*m.events = append(*m.events, "kill:"+name)
	}
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

func (m *mockCheckoutLifecycle) IsAvailable() bool {
	return m.available
}

func (m *mockCheckoutLifecycle) ResolvePath(_ context.Context, checkoutPath string) (worktrunk.Snapshot, error) {
	m.resolveCalls = append(m.resolveCalls, checkoutPath)
	if snapshot, ok := m.resolveSnapshots[checkoutPath]; ok {
		return snapshot, nil
	}
	return worktrunk.Snapshot{
		RepoRoot:      checkoutPath,
		CheckoutPath:  checkoutPath,
		Branch:        "main",
		BaseBranch:    "main",
		DefaultBranch: "main",
	}, nil
}

func (m *mockCheckoutLifecycle) Create(_ context.Context, repoPath, branch, base string) (worktrunk.Snapshot, error) {
	m.createCalls = append(m.createCalls, mockWorktreeCreateCall{RepoPath: repoPath, Branch: branch, Base: base})
	if m.createErr != nil {
		return worktrunk.Snapshot{}, m.createErr
	}
	if snapshot, ok := m.createSnapshots[branch]; ok {
		return snapshot, nil
	}
	if base == "" {
		base = "main"
	}
	return worktrunk.Snapshot{
		RepoRoot:      repoPath,
		CheckoutPath:  filepath.Join(repoPath, ".worktrees", branch),
		Branch:        branch,
		BaseBranch:    base,
		DefaultBranch: "main",
	}, nil
}

func (m *mockCheckoutLifecycle) Remove(_ context.Context, repoPath string, branches ...string) error {
	m.removeCalls = append(m.removeCalls, mockWorktreeRemoveCall{RepoPath: repoPath, Branches: append([]string(nil), branches...)})
	if m.events != nil {
		*m.events = append(*m.events, "remove:"+strings.Join(branches, ","))
	}
	return m.removeErr
}

func (m *mockCheckoutLifecycle) Merge(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockGitInspector) RepoRoot(_ context.Context, path string) (string, error) {
	if root, ok := m.repoRoots[path]; ok {
		return root, nil
	}
	return path, nil
}

func (m *mockGitInspector) CurrentBranch(_ context.Context, path string) (string, error) {
	if branch, ok := m.currentBranches[path]; ok {
		return branch, nil
	}
	return "main", nil
}

func (m *mockGitInspector) DefaultBranch(_ context.Context, path string) (string, error) {
	if branch, ok := m.defaultBranches[path]; ok {
		return branch, nil
	}
	return "main", nil
}

func (m *mockGitInspector) MergeBase(_ context.Context, path string, left string, right string) (string, error) {
	if sha, ok := m.mergeBases[fmt.Sprintf("%s|%s|%s", path, left, right)]; ok {
		return sha, nil
	}
	return "", nil
}

func newTestManager(store *Store, sessions *mockSessionCreator) (*Manager, *mockCheckoutLifecycle, *mockGitInspector) {
	checkouts := &mockCheckoutLifecycle{
		available:        true,
		resolveSnapshots: make(map[string]worktrunk.Snapshot),
		createSnapshots:  make(map[string]worktrunk.Snapshot),
	}
	git := &mockGitInspector{
		repoRoots:       make(map[string]string),
		currentBranches: make(map[string]string),
		defaultBranches: make(map[string]string),
		mergeBases:      make(map[string]string),
	}
	mgr := NewManager(store, sessions, checkouts, "colo-")
	mgr.git = git
	return mgr, checkouts, git
}

func TestManagerCreate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr, _, _ := newTestManager(store, mock)

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

func TestManagerCreateRejectsDashboardTitle(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr, _, _ := newTestManager(store, mock)

	if _, err := mgr.Create(context.Background(), "dashboard", agent.Claude, "/tmp/project", "main", LayoutAgent); err == nil {
		t.Fatal("expected reserved dashboard title error")
	}

	if len(mock.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(mock.createCalls))
	}
}

func TestManagerDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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

func TestManagerDeleteManagedCheckoutKeepsRuntimeWhenRemoveFails(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	mgr, checkouts, _ := newTestManager(store, sessions)

	ws, err := mgr.CreateWithWorktree(context.Background(), ManagedWorkspaceRequest{
		Title:      "managed-remove-fail",
		AgentType:  agent.Codex,
		RepoRoot:   "/repo",
		Branch:     "feature-remove",
		BaseBranch: "main",
		Layout:     LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateWithWorktree: %v", err)
	}

	checkouts.removeErr = errors.New("dirty worktree")
	if err := mgr.Delete(context.Background(), ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if len(sessions.killCalls) != 0 {
		t.Fatalf("kill calls = %v, want none", sessions.killCalls)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Workspaces) != 1 {
		t.Fatalf("workspaces = %d, want 1", len(state.Workspaces))
	}
	if len(state.Checkouts) != 1 {
		t.Fatalf("checkouts = %d, want 1", len(state.Checkouts))
	}
}

func TestManagerDeleteManagedCheckoutRetriesWhenCheckoutAlreadyGone(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{killErr: errors.New("tmux unavailable")}
	mgr, checkouts, _ := newTestManager(store, sessions)

	ws, err := mgr.CreateWithWorktree(context.Background(), ManagedWorkspaceRequest{
		Title:      "managed-kill-fail",
		AgentType:  agent.Codex,
		RepoRoot:   "/repo",
		Branch:     "feature-kill",
		BaseBranch: "main",
		Layout:     LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateWithWorktree: %v", err)
	}

	if err := mgr.Delete(context.Background(), ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if len(checkouts.removeCalls) != 1 {
		t.Fatalf("remove calls = %v, want one successful removal before kill failure", checkouts.removeCalls)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Workspaces) != 1 {
		t.Fatalf("workspaces = %d, want 1", len(state.Workspaces))
	}
	if len(state.Checkouts) != 1 {
		t.Fatalf("checkouts = %d, want 1", len(state.Checkouts))
	}

	sessions.killErr = nil
	checkouts.removeErr = &worktrunk.CommandError{Binary: "wt", Args: []string{"remove"}, Stderr: "worktree not found"}

	if err := mgr.Delete(context.Background(), ws.ID); err != nil {
		t.Fatalf("retry Delete: %v", err)
	}

	state, err = store.LoadState()
	if err != nil {
		t.Fatalf("LoadState after retry: %v", err)
	}
	if len(state.Workspaces) != 0 {
		t.Fatalf("workspaces after retry = %d, want 0", len(state.Workspaces))
	}
	if len(state.Checkouts) != 0 {
		t.Fatalf("checkouts after retry = %d, want 0", len(state.Checkouts))
	}
}

func TestManagerList(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)
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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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
	mgr, _, _ := newTestManager(store, mock)

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

func TestManagerCreateWithWorktreePersistsManagedCheckout(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	events := []string{}
	sessions := &mockSessionCreator{events: &events}
	mgr, checkouts, _ := newTestManager(store, sessions)
	checkouts.events = &events

	ws, err := mgr.CreateWithWorktree(context.Background(), ManagedWorkspaceRequest{
		Title:      "managed",
		AgentType:  agent.Claude,
		RepoRoot:   "/repo",
		Branch:     "feature-auth",
		BaseBranch: "main",
		Layout:     LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateWithWorktree: %v", err)
	}

	if ws.Branch != "feature-auth" {
		t.Fatalf("workspace branch = %q, want feature-auth", ws.Branch)
	}
	if ws.CheckoutOwnership != OwnershipColosseumManaged {
		t.Fatalf("workspace ownership = %q, want %q", ws.CheckoutOwnership, OwnershipColosseumManaged)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(state.Repositories))
	}
	if len(state.Checkouts) != 1 {
		t.Fatalf("checkouts = %d, want 1", len(state.Checkouts))
	}
	if state.Checkouts[0].Ownership != OwnershipColosseumManaged {
		t.Fatalf("checkout ownership = %q, want managed", state.Checkouts[0].Ownership)
	}

	if err := mgr.Delete(context.Background(), ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(checkouts.removeCalls) != 1 {
		t.Fatalf("remove calls = %d, want 1", len(checkouts.removeCalls))
	}
	if got := checkouts.removeCalls[0].Branches; len(got) != 1 || got[0] != "feature-auth" {
		t.Fatalf("remove branches = %v, want [feature-auth]", got)
	}
	if len(events) < 2 || events[0] != "remove:feature-auth" || events[1] != "kill:"+ws.SessionName {
		t.Fatalf("events = %v, want remove before kill", events)
	}
}

func TestManagerCreateWithWorktreeGeneratesUniqueBranchesAcrossRecreate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	mgr, checkouts, _ := newTestManager(store, sessions)

	first, err := mgr.CreateWithWorktree(context.Background(), ManagedWorkspaceRequest{
		Title:     "Repeated Workspace",
		AgentType: agent.Codex,
		RepoRoot:  "/repo",
		Layout:    LayoutAgent,
	})
	if err != nil {
		t.Fatalf("first CreateWithWorktree: %v", err)
	}
	if err := mgr.Delete(context.Background(), first.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = mgr.CreateWithWorktree(context.Background(), ManagedWorkspaceRequest{
		Title:     "Repeated Workspace",
		AgentType: agent.Codex,
		RepoRoot:  "/repo",
		Layout:    LayoutAgent,
	})
	if err != nil {
		t.Fatalf("second CreateWithWorktree: %v", err)
	}

	if len(checkouts.createCalls) != 2 {
		t.Fatalf("create calls = %d, want 2", len(checkouts.createCalls))
	}
	if checkouts.createCalls[0].Branch == checkouts.createCalls[1].Branch {
		t.Fatalf("generated branch reused across recreate: %q", checkouts.createCalls[0].Branch)
	}
	if !strings.HasPrefix(checkouts.createCalls[0].Branch, "feat-repeated-workspace-") {
		t.Fatalf("first generated branch = %q, want feat-repeated-workspace-*", checkouts.createCalls[0].Branch)
	}
	if !strings.HasPrefix(checkouts.createCalls[1].Branch, "feat-repeated-workspace-") {
		t.Fatalf("second generated branch = %q, want feat-repeated-workspace-*", checkouts.createCalls[1].Branch)
	}
}

func TestManagerCreateExperimentCreatesCandidatesAndBroadcasts(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	mgr, _, _ := newTestManager(store, sessions)

	result, err := mgr.CreateExperiment(context.Background(), ExperimentRequest{
		Title:         "Auth fix",
		Prompt:        "ship the fix",
		RepoRoot:      "/repo",
		BaseBranch:    "main",
		AgentStrategy: ExperimentAgentAllSupported,
		AgentType:     agent.Claude,
		Layout:        LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateExperiment: %v", err)
	}

	if got := len(result.Workspaces); got != len(agent.Supported()) {
		t.Fatalf("workspaces = %d, want %d", got, len(agent.Supported()))
	}
	if result.Experiment == nil {
		t.Fatal("expected experiment metadata")
	}
	if result.Experiment.Status != ExperimentRunning {
		t.Fatalf("experiment status = %q, want %q", result.Experiment.Status, ExperimentRunning)
	}
	if len(result.Broadcast.Delivered) != len(agent.Supported()) {
		t.Fatalf("broadcast delivered = %d, want %d", len(result.Broadcast.Delivered), len(agent.Supported()))
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Experiments) != 1 {
		t.Fatalf("experiments = %d, want 1", len(state.Experiments))
	}
	if len(state.Checkouts) != len(agent.Supported()) {
		t.Fatalf("checkouts = %d, want %d", len(state.Checkouts), len(agent.Supported()))
	}
}

func TestManagerCreateExperimentGeneratesUniqueBranchesAcrossRecreate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	mgr, checkouts, _ := newTestManager(store, sessions)

	first, err := mgr.CreateExperiment(context.Background(), ExperimentRequest{
		Title:         "Auth fix",
		RepoRoot:      "/repo",
		BaseBranch:    "main",
		AgentStrategy: ExperimentAgentAllSupported,
		AgentType:     agent.Claude,
		Layout:        LayoutAgent,
	})
	if err != nil {
		t.Fatalf("first CreateExperiment: %v", err)
	}
	for _, ws := range first.Workspaces {
		if err := mgr.Delete(context.Background(), ws.ID); err != nil {
			t.Fatalf("Delete(%s): %v", ws.ID, err)
		}
	}

	second, err := mgr.CreateExperiment(context.Background(), ExperimentRequest{
		Title:         "Auth fix",
		RepoRoot:      "/repo",
		BaseBranch:    "main",
		AgentStrategy: ExperimentAgentAllSupported,
		AgentType:     agent.Claude,
		Layout:        LayoutAgent,
	})
	if err != nil {
		t.Fatalf("second CreateExperiment: %v", err)
	}
	_ = second

	perRun := len(agent.Supported())
	if len(checkouts.createCalls) != perRun*2 {
		t.Fatalf("create calls = %d, want %d", len(checkouts.createCalls), perRun*2)
	}
	for i := 0; i < perRun; i++ {
		firstBranch := checkouts.createCalls[i].Branch
		secondBranch := checkouts.createCalls[perRun+i].Branch
		if firstBranch == secondBranch {
			t.Fatalf("candidate %d reused generated branch %q across recreate", i, firstBranch)
		}
		if !strings.HasPrefix(firstBranch, "exp-auth-fix-") {
			t.Fatalf("first branch = %q, want exp-auth-fix-*", firstBranch)
		}
		if !strings.HasPrefix(secondBranch, "exp-auth-fix-") {
			t.Fatalf("second branch = %q, want exp-auth-fix-*", secondBranch)
		}
	}
}

func TestManagerCreateStandaloneDoesNotRequireCheckoutLifecycle(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	git := &mockGitInspector{
		repoRoots: map[string]string{
			"/repo/subdir": "/repo",
		},
		currentBranches: map[string]string{
			"/repo/subdir": "/repo-branch",
		},
		defaultBranches: map[string]string{
			"/repo": "main",
		},
		mergeBases: map[string]string{
			"/repo|/repo-branch|main": "abc123",
		},
	}

	mgr := NewManager(store, sessions, nil, "colo-")
	mgr.git = git

	ws, err := mgr.CreateStandalone(context.Background(), StandaloneWorkspaceRequest{
		Title:        "standalone",
		AgentType:    agent.Claude,
		CheckoutPath: "/repo/subdir",
		Layout:       LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateStandalone: %v", err)
	}
	if ws.ProjectPath != "/repo/subdir" {
		t.Fatalf("project path = %q, want /repo/subdir", ws.ProjectPath)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Checkouts) != 1 {
		t.Fatalf("checkouts = %d, want 1", len(state.Checkouts))
	}
	if state.Checkouts[0].CheckoutPath != "/repo/subdir" {
		t.Fatalf("checkout path = %q, want /repo/subdir", state.Checkouts[0].CheckoutPath)
	}
	if state.Checkouts[0].RepoRoot != "/repo" {
		t.Fatalf("repo root = %q, want /repo", state.Checkouts[0].RepoRoot)
	}
	if state.Repositories[0].WorktrunkAvailable {
		t.Fatal("expected worktrunk availability to stay false without checkout lifecycle")
	}
}

func TestManagerCreateStandaloneSkipsCheckoutResolution(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	sessions := &mockSessionCreator{}
	mgr, checkouts, git := newTestManager(store, sessions)

	git.repoRoots["/repo"] = "/repo"
	git.currentBranches["/repo"] = "main"
	git.defaultBranches["/repo"] = "main"

	if _, err := mgr.CreateStandalone(context.Background(), StandaloneWorkspaceRequest{
		Title:        "standalone",
		AgentType:    agent.Claude,
		CheckoutPath: "/repo",
		Layout:       LayoutAgent,
	}); err != nil {
		t.Fatalf("CreateStandalone: %v", err)
	}
	if len(checkouts.resolveCalls) != 0 {
		t.Fatalf("resolve calls = %v, want none", checkouts.resolveCalls)
	}
}
