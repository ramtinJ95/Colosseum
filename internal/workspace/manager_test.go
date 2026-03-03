package workspace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
)

type mockSessionCreator struct {
	createCalls []mockCreateCall
	killCalls   []string
	splitCalls  []mockSplitCall
	switchCalls []string
	splitCount  int
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

func (m *mockSessionCreator) CreateSession(_ context.Context, name string, startDir string) error {
	m.createCalls = append(m.createCalls, mockCreateCall{Name: name, StartDir: startDir})
	return nil
}

func (m *mockSessionCreator) KillSession(_ context.Context, name string) error {
	m.killCalls = append(m.killCalls, name)
	return nil
}

func (m *mockSessionCreator) SplitWindow(_ context.Context, session string, horizontal bool, startDir string) (string, error) {
	m.splitCalls = append(m.splitCalls, mockSplitCall{
		Session:    session,
		Horizontal: horizontal,
		StartDir:   startDir,
	})
	m.splitCount++
	return "%1", nil
}

func (m *mockSessionCreator) SwitchClient(_ context.Context, name string) error {
	m.switchCalls = append(m.switchCalls, name)
	return nil
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

func TestManagerList(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))
	mock := &mockSessionCreator{}
	mgr := NewManager(store, mock, "colo-")

	ctx := context.Background()
	if _, err := mgr.Create(ctx, "ws-1", agent.Claude, "/tmp/p1", "main", LayoutAgent); err != nil {
		t.Fatalf("Create ws-1: %v", err)
	}
	if _, err := mgr.Create(ctx, "ws-2", agent.Gemini, "/tmp/p2", "dev", LayoutAgentShell); err != nil {
		t.Fatalf("Create ws-2: %v", err)
	}
	if _, err := mgr.Create(ctx, "ws-3", agent.Aider, "/tmp/p3", "feature", LayoutAgentShellLogs); err != nil {
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
