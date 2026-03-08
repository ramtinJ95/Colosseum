package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() ([]Workspace, error) {
	state, err := s.LoadState()
	if err != nil {
		return nil, err
	}
	return state.Workspaces, nil
}

func (s *Store) Save(workspaces []Workspace) error {
	return s.UpdateState(func(state *State) error {
		state.Workspaces = workspaces
		return nil
	})
}

func (s *Store) LoadState() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadStateUnsafe()
}

func (s *Store) SaveState(state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveStateUnsafe(state)
}

func (s *Store) UpdateState(update func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadStateUnsafe()
	if err != nil {
		return err
	}
	if err := update(&state); err != nil {
		return err
	}
	return s.saveStateUnsafe(state)
}

func (s *Store) Add(ws Workspace) error {
	return s.UpdateState(func(state *State) error {
		state.Workspaces = append(state.Workspaces, ws)
		return nil
	})
}

func (s *Store) Remove(id string) error {
	return s.UpdateState(func(state *State) error {
		filtered := make([]Workspace, 0, len(state.Workspaces))
		for _, ws := range state.Workspaces {
			if ws.ID != id {
				filtered = append(filtered, ws)
			}
		}
		state.Workspaces = filtered
		return nil
	})
}

func (s *Store) Update(ws Workspace) error {
	return s.UpdateState(func(state *State) error {
		for i, existing := range state.Workspaces {
			if existing.ID == ws.ID {
				state.Workspaces[i] = ws
				return nil
			}
		}
		return fmt.Errorf("workspace %q not found", ws.ID)
	})
}

func (s *Store) Get(id string) (Workspace, bool, error) {
	state, err := s.LoadState()
	if err != nil {
		return Workspace{}, false, fmt.Errorf("loading workspaces: %w", err)
	}
	for _, ws := range state.Workspaces {
		if ws.ID == id {
			return ws, true, nil
		}
	}
	return Workspace{}, false, nil
}

func (s *Store) List() ([]Workspace, error) {
	return s.Load()
}

func (s *Store) loadStateUnsafe() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("reading %s: %w", s.path, err)
	}

	if len(data) == 0 {
		return State{}, nil
	}

	switch firstJSONToken(data) {
	case '[':
		var workspaces []Workspace
		if err := json.Unmarshal(data, &workspaces); err != nil {
			return State{}, fmt.Errorf("parsing %s: %w", s.path, err)
		}
		return migrateLegacyWorkspaces(workspaces), nil
	default:
		var state State
		if err := json.Unmarshal(data, &state); err != nil {
			return State{}, fmt.Errorf("parsing %s: %w", s.path, err)
		}
		return state, nil
	}
}

func (s *Store) saveStateUnsafe(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", s.path, err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
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

func firstJSONToken(data []byte) byte {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return b
		}
	}
	return 0
}

func migrateLegacyWorkspaces(workspaces []Workspace) State {
	state := State{
		Workspaces: make([]Workspace, 0, len(workspaces)),
	}

	repositoryIDs := make(map[string]string)
	checkoutIDs := make(map[string]string)

	for _, ws := range workspaces {
		repoRoot := ws.ProjectPath
		repositoryID, ok := repositoryIDs[repoRoot]
		if !ok {
			repositoryID = stableStateID("repository", repoRoot)
			repositoryIDs[repoRoot] = repositoryID
			state.Repositories = append(state.Repositories, Repository{
				ID:                 repositoryID,
				RootPath:           repoRoot,
				DefaultBranch:      "",
				Backend:            BackendExternal,
				WorktrunkAvailable: false,
				CreatedAt:          fallbackCreatedAt(ws.CreatedAt),
			})
		}

		checkoutKey := repoRoot + "\x00" + ws.ProjectPath + "\x00" + ws.Branch
		checkoutID, ok := checkoutIDs[checkoutKey]
		if !ok {
			checkoutID = stableStateID("checkout", checkoutKey)
			checkoutIDs[checkoutKey] = checkoutID
			state.Checkouts = append(state.Checkouts, Checkout{
				ID:            checkoutID,
				RepositoryID:  repositoryID,
				RepoRoot:      repoRoot,
				CheckoutPath:  ws.ProjectPath,
				Branch:        ws.Branch,
				DefaultBranch: "",
				Backend:       BackendExternal,
				Ownership:     OwnershipAttached,
				CreatedFrom:   CreatedFromStandalone,
				CreatedAt:     fallbackCreatedAt(ws.CreatedAt),
			})
		}

		ws.RepositoryID = repositoryID
		ws.CheckoutID = checkoutID
		ws.CheckoutBackend = BackendExternal
		ws.CheckoutOwnership = OwnershipAttached
		state.Workspaces = append(state.Workspaces, ws)
	}

	return state
}

func stableStateID(kind string, value string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(kind+"\x00"+value)).String()
}

func fallbackCreatedAt(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
