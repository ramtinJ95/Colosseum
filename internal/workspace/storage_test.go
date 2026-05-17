package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAddAndLoad(t *testing.T) {
	store := newTestStore(t)

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
	store := newTestStore(t)

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

func TestStoreRemoveDeletesAgentStatusReports(t *testing.T) {
	store := newTestStore(t)
	if err := store.SaveState(State{
		Workspaces: []Workspace{
			newTestWorkspace("id-1", "ws-1"),
			newTestWorkspace("id-2", "ws-2"),
		},
		AgentStatusReports: []AgentStatusReport{
			{WorkspaceID: "id-1", Pane: "agent", Status: "Working"},
			{WorkspaceID: "id-2", Pane: "agent", Status: "Idle"},
		},
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	if err := store.Remove("id-1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.AgentStatusReports) != 1 || state.AgentStatusReports[0].WorkspaceID != "id-2" {
		t.Fatalf("reports = %+v, want only id-2", state.AgentStatusReports)
	}
}

func TestStoreUpdate(t *testing.T) {
	store := newTestStore(t)

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
	store, storePath := newTestStoreWithPath(t)

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
	store := NewStore(filepath.Join(t.TempDir(), "nonexistent.json"))

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected empty slice, got %d workspaces", len(loaded))
	}
}

func TestStoreLoadsLegacyWorkspaceArray(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "workspaces.json")

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
	assertStateCounts(t, state, 1, 1, 1, 0)
	if got := state.Workspaces[0].Title; got != "legacy" {
		t.Fatalf("title = %q, want legacy", got)
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
