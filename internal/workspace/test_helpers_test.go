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
	renameCalls   []mockRenameCall
	events        *[]string
	createCount   int
	splitCount    int
	createErr     error
	killErr       error
	splitErrAt    int
	sendKeysErr   error
	sendKeysErrs  map[string]error
	renameErr     error
}

type mockRenameCall struct {
	OldName string
	NewName string
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

func (m *mockSessionCreator) RenameSession(_ context.Context, oldName, newName string) error {
	m.renameCalls = append(m.renameCalls, mockRenameCall{OldName: oldName, NewName: newName})
	return m.renameErr
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(filepath.Join(t.TempDir(), "workspaces.json"))
}

func newTestStoreWithPath(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspaces.json")
	return NewStore(path), path
}

func newTestWorkspace(id, title string) Workspace {
	return Workspace{
		ID:          id,
		Title:       title,
		AgentType:   agent.Claude,
		ProjectPath: "/tmp/test",
		Branch:      "main",
		Layout:      LayoutAgent,
		Status:      agent.StatusIdle,
		SessionName: "colo-" + title,
		PaneTargets: map[string]string{"agent": "colo-" + title + ":0.0"},
		CreatedAt:   time.Now(),
	}
}

type managerHarness struct {
	ctx       context.Context
	store     *Store
	mgr       *Manager
	sessions  *mockSessionCreator
	checkouts *mockCheckoutLifecycle
	git       *mockGitInspector
}

func newManagerHarness(t *testing.T) *managerHarness {
	t.Helper()
	return newManagerHarnessWithStore(t, newTestStore(t))
}

func newManagerHarnessWithStore(t *testing.T, store *Store) *managerHarness {
	t.Helper()
	sessions := &mockSessionCreator{}
	mgr, checkouts, git := newTestManager(store, sessions)
	return &managerHarness{
		ctx:       context.Background(),
		store:     store,
		mgr:       mgr,
		sessions:  sessions,
		checkouts: checkouts,
		git:       git,
	}
}

func (h *managerHarness) mustCreate(t *testing.T, title string, agentType agent.AgentType, projectPath string, branch string, layout LayoutType) *Workspace {
	t.Helper()
	ws, err := h.mgr.Create(h.ctx, title, agentType, projectPath, branch, layout)
	if err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	return ws
}

func (h *managerHarness) mustCreateManaged(t *testing.T, req ManagedWorkspaceRequest) *Workspace {
	t.Helper()
	ws, err := h.mgr.CreateWithWorktree(h.ctx, req)
	if err != nil {
		t.Fatalf("CreateWithWorktree(%q): %v", req.Title, err)
	}
	return ws
}

func (h *managerHarness) mustCreateStandalone(t *testing.T, req StandaloneWorkspaceRequest) *Workspace {
	t.Helper()
	ws, err := h.mgr.CreateStandalone(h.ctx, req)
	if err != nil {
		t.Fatalf("CreateStandalone(%q): %v", req.Title, err)
	}
	return ws
}

func (h *managerHarness) mustCreateExperiment(t *testing.T, req ExperimentRequest) *ExperimentCreateResult {
	t.Helper()
	result, err := h.mgr.CreateExperiment(h.ctx, req)
	if err != nil {
		t.Fatalf("CreateExperiment(%q): %v", req.Title, err)
	}
	return result
}

func (h *managerHarness) mustList(t *testing.T) []Workspace {
	t.Helper()
	workspaces, err := h.mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return workspaces
}

func (h *managerHarness) mustState(t *testing.T) State {
	t.Helper()
	state, err := h.store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	return state
}

func (h *managerHarness) resetSendKeys() {
	h.sessions.sendKeysCalls = nil
	h.sessions.sendKeysErr = nil
	h.sessions.sendKeysErrs = nil
}

func (h *managerHarness) mustCreateBroadcastPair(t *testing.T) (*Workspace, *Workspace) {
	t.Helper()
	claude := h.mustCreate(t, "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent)
	codex := h.mustCreate(t, "ws-2", agent.Codex, "/tmp/p2", "main", LayoutAgent)
	h.resetSendKeys()
	return claude, codex
}

func defaultManagedWorkspaceRequest(title string, agentType agent.AgentType) ManagedWorkspaceRequest {
	return ManagedWorkspaceRequest{
		Title:      title,
		AgentType:  agentType,
		RepoRoot:   "/repo",
		BaseBranch: "main",
		Layout:     LayoutAgent,
	}
}

func defaultExperimentRequest(title string) ExperimentRequest {
	return ExperimentRequest{
		Title:         title,
		RepoRoot:      "/repo",
		BaseBranch:    "main",
		AgentStrategy: ExperimentAgentAllSupported,
		AgentType:     agent.Claude,
		Layout:        LayoutAgent,
	}
}

func assertStateCounts(t *testing.T, state State, wantWorkspaces, wantRepositories, wantCheckouts, wantExperiments int) {
	t.Helper()
	if got := len(state.Workspaces); got != wantWorkspaces {
		t.Fatalf("workspaces = %d, want %d", got, wantWorkspaces)
	}
	if got := len(state.Repositories); got != wantRepositories {
		t.Fatalf("repositories = %d, want %d", got, wantRepositories)
	}
	if got := len(state.Checkouts); got != wantCheckouts {
		t.Fatalf("checkouts = %d, want %d", got, wantCheckouts)
	}
	if got := len(state.Experiments); got != wantExperiments {
		t.Fatalf("experiments = %d, want %d", got, wantExperiments)
	}
}
