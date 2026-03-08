package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
)

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

func TestStoreAddAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))

	ws := newTestWorkspace("id-1", "test-ws")
	if err := store.Add(ws); err != nil {
		t.Fatalf("Add: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(loaded))
	}
	if loaded[0].ID != "id-1" {
		t.Errorf("expected ID %q, got %q", "id-1", loaded[0].ID)
	}
	if loaded[0].Title != "test-ws" {
		t.Errorf("expected title %q, got %q", "test-ws", loaded[0].Title)
	}
}

func TestStoreRemove(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))

	ws1 := newTestWorkspace("id-1", "ws-1")
	ws2 := newTestWorkspace("id-2", "ws-2")
	if err := store.Add(ws1); err != nil {
		t.Fatalf("Add ws1: %v", err)
	}
	if err := store.Add(ws2); err != nil {
		t.Fatalf("Add ws2: %v", err)
	}

	if err := store.Remove("id-1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 workspace after remove, got %d", len(loaded))
	}
	if loaded[0].ID != "id-2" {
		t.Errorf("expected remaining workspace ID %q, got %q", "id-2", loaded[0].ID)
	}
}

func TestStoreUpdate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "workspaces.json"))

	ws := newTestWorkspace("id-1", "original-title")
	if err := store.Add(ws); err != nil {
		t.Fatalf("Add: %v", err)
	}

	ws.Title = "updated-title"
	if err := store.Update(ws); err != nil {
		t.Fatalf("Update: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(loaded))
	}
	if loaded[0].Title != "updated-title" {
		t.Errorf("expected title %q, got %q", "updated-title", loaded[0].Title)
	}
}

func TestStoreAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "workspaces.json")
	store := NewStore(storePath)

	ws := newTestWorkspace("id-1", "test-ws")
	if err := store.Add(ws); err != nil {
		t.Fatalf("Add: %v", err)
	}

	tmpPath := storePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("expected .tmp file to not persist after write")
	}

	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Error("expected final file to exist")
	}
}

func TestStoreEmptyFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "nonexistent.json"))

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty slice, got %d workspaces", len(loaded))
	}
}

func TestStoreLoadsLegacyWorkspaceArray(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "workspaces.json")

	legacy := []Workspace{newTestWorkspace("id-1", "legacy")}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(storePath, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewStore(storePath)
	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.Workspaces) != 1 {
		t.Fatalf("workspaces = %d, want 1", len(state.Workspaces))
	}
	if got := state.Workspaces[0].Title; got != "legacy" {
		t.Fatalf("title = %q, want legacy", got)
	}
	if len(state.Repositories) != 1 {
		t.Fatalf("repositories = %d, want 1", len(state.Repositories))
	}
	if len(state.Checkouts) != 1 {
		t.Fatalf("checkouts = %d, want 1", len(state.Checkouts))
	}
	if state.Workspaces[0].RepositoryID == "" {
		t.Fatal("expected migrated workspace repository id")
	}
	if state.Workspaces[0].CheckoutID == "" {
		t.Fatal("expected migrated workspace checkout id")
	}
	if state.Checkouts[0].Ownership != OwnershipAttached {
		t.Fatalf("checkout ownership = %q, want %q", state.Checkouts[0].Ownership, OwnershipAttached)
	}
}
