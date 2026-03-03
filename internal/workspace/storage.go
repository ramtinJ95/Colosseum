package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() ([]Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadUnsafe()
}

func (s *Store) Save(workspaces []Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveUnsafe(workspaces)
}

func (s *Store) Add(ws Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.loadUnsafe()
	if err != nil {
		return fmt.Errorf("loading workspaces: %w", err)
	}
	workspaces = append(workspaces, ws)
	return s.saveUnsafe(workspaces)
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.loadUnsafe()
	if err != nil {
		return fmt.Errorf("loading workspaces: %w", err)
	}

	filtered := make([]Workspace, 0, len(workspaces))
	for _, ws := range workspaces {
		if ws.ID != id {
			filtered = append(filtered, ws)
		}
	}

	return s.saveUnsafe(filtered)
}

func (s *Store) Update(ws Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.loadUnsafe()
	if err != nil {
		return fmt.Errorf("loading workspaces: %w", err)
	}

	found := false
	for i, existing := range workspaces {
		if existing.ID == ws.ID {
			workspaces[i] = ws
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("workspace %q not found", ws.ID)
	}

	return s.saveUnsafe(workspaces)
}

func (s *Store) Get(id string) (Workspace, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workspaces, err := s.loadUnsafe()
	if err != nil {
		return Workspace{}, false, fmt.Errorf("loading workspaces: %w", err)
	}

	for _, ws := range workspaces {
		if ws.ID == id {
			return ws, true, nil
		}
	}

	return Workspace{}, false, nil
}

func (s *Store) List() ([]Workspace, error) {
	return s.Load()
}

func (s *Store) loadUnsafe() ([]Workspace, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Workspace{}, nil
		}
		return nil, fmt.Errorf("reading %s: %w", s.path, err)
	}

	if len(data) == 0 {
		return []Workspace{}, nil
	}

	var workspaces []Workspace
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", s.path, err)
	}

	return workspaces, nil
}

func (s *Store) saveUnsafe(workspaces []Workspace) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", s.path, err)
	}

	data, err := json.MarshalIndent(workspaces, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling workspaces: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmpPath, s.path, err)
	}

	return nil
}
