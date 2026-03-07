package notification

import (
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
)

func TestStoreAddStatusTransitionTracksUnread(t *testing.T) {
	store := NewStore()

	entry := store.AddStatusTransition("ws-1", "alpha", agent.StatusWorking, agent.StatusWaiting, false)
	if entry.Message != "alpha needs attention" {
		t.Fatalf("message = %q, want %q", entry.Message, "alpha needs attention")
	}
	if got := store.UnreadCount("ws-1"); got != 1 {
		t.Fatalf("UnreadCount = %d, want 1", got)
	}
}

func TestStoreMarkWorkspaceRead(t *testing.T) {
	store := NewStore()
	store.AddStatusTransition("ws-1", "alpha", agent.StatusWorking, agent.StatusWaiting, false)
	store.AddStatusTransition("ws-1", "alpha", agent.StatusWaiting, agent.StatusError, false)
	store.AddStatusTransition("ws-2", "beta", agent.StatusWorking, agent.StatusIdle, false)

	updated := store.MarkWorkspaceRead("ws-1")
	if updated != 2 {
		t.Fatalf("updated = %d, want 2", updated)
	}
	if got := store.UnreadCount("ws-1"); got != 0 {
		t.Fatalf("UnreadCount(ws-1) = %d, want 0", got)
	}
	if got := store.UnreadCount("ws-2"); got != 1 {
		t.Fatalf("UnreadCount(ws-2) = %d, want 1", got)
	}
}

func TestStoreAddStatusTransitionCanStartRead(t *testing.T) {
	store := NewStore()
	store.AddStatusTransition("ws-1", "alpha", agent.StatusWorking, agent.StatusIdle, true)

	if got := store.UnreadCount("ws-1"); got != 0 {
		t.Fatalf("UnreadCount = %d, want 0", got)
	}
}
