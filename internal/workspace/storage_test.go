package workspace

import (
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
