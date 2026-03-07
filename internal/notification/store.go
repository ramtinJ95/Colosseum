package notification

import (
	"sync"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
)

type Entry struct {
	WorkspaceID    string
	WorkspaceTitle string
	Previous       agent.Status
	Current        agent.Status
	Message        string
	CreatedAt      time.Time
	Read           bool
}

type Store struct {
	mu      sync.RWMutex
	entries []Entry
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) AddStatusTransition(workspaceID, workspaceTitle string, previous, current agent.Status, read bool) Entry {
	entry := Entry{
		WorkspaceID:    workspaceID,
		WorkspaceTitle: workspaceTitle,
		Previous:       previous,
		Current:        current,
		Message:        transitionMessage(workspaceTitle, current),
		CreatedAt:      time.Now(),
		Read:           read,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	return entry
}

func (s *Store) MarkWorkspaceRead(workspaceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var updated int
	for i := range s.entries {
		if s.entries[i].WorkspaceID == workspaceID && !s.entries[i].Read {
			s.entries[i].Read = true
			updated++
		}
	}
	return updated
}

func (s *Store) UnreadCount(workspaceID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	for _, entry := range s.entries {
		if entry.WorkspaceID == workspaceID && !entry.Read {
			count++
		}
	}
	return count
}

func (s *Store) List() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Entry, len(s.entries))
	copy(items, s.entries)
	return items
}

func transitionMessage(workspaceTitle string, current agent.Status) string {
	switch current {
	case agent.StatusWaiting:
		return workspaceTitle + " needs attention"
	case agent.StatusIdle:
		return workspaceTitle + " is idle"
	case agent.StatusError:
		return workspaceTitle + " hit an error"
	case agent.StatusStopped:
		return workspaceTitle + " stopped"
	default:
		return workspaceTitle + " changed status"
	}
}
