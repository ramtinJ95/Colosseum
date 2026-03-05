package status

import (
	"context"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type mockCapturer struct {
	content string
	err     error
}

func (m *mockCapturer) CapturePane(_ context.Context, _ string, _ int) (string, error) {
	return m.content, m.err
}

type mockProvider struct {
	workspaces []workspace.Workspace
}

func (m *mockProvider) List() ([]workspace.Workspace, error) {
	return m.workspaces, nil
}

func TestPollerDetectsStatusChange(t *testing.T) {
	capturer := &mockCapturer{content: "⠹ Working (esc to interrupt)"}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	select {
	case update := <-poller.Updates():
		if update.WorkspaceID != "ws-1" {
			t.Errorf("got workspace ID %q, want ws-1", update.WorkspaceID)
		}
		if update.Current != agent.StatusWorking {
			t.Errorf("got status %s, want Working", update.Current)
		}
		if update.Previous != agent.StatusUnknown {
			t.Errorf("got previous %s, want Unknown", update.Previous)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for status update")
	}
}

func TestPollerNoUpdateWhenStatusUnchanged(t *testing.T) {
	capturer := &mockCapturer{content: "⠹ Working (esc to interrupt)"}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	// First update should come through
	<-poller.Updates()

	// Second poll with same content should NOT produce an update
	select {
	case <-poller.Updates():
		t.Error("unexpected second update when status unchanged")
	case <-time.After(200 * time.Millisecond):
		// Expected: no update
	}
}

func TestPollerStatusTransition(t *testing.T) {
	capturer := &mockCapturer{content: "⠹ Working (esc to interrupt)"}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go poller.Run(ctx)

	// First update: Unknown -> Working
	update := <-poller.Updates()
	if update.Current != agent.StatusWorking {
		t.Fatalf("first update: got %s, want Working", update.Current)
	}

	// Change capturer content to idle
	capturer.content = ">\n"

	// Second update: Working -> Idle
	update = <-poller.Updates()
	if update.Previous != agent.StatusWorking {
		t.Errorf("second update previous: got %s, want Working", update.Previous)
	}
	if update.Current != agent.StatusIdle {
		t.Errorf("second update current: got %s, want Idle", update.Current)
	}
}

func TestPollerCurrentStatus(t *testing.T) {
	capturer := &mockCapturer{content: "⠹ Working (esc to interrupt)"}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	<-poller.Updates()

	got := poller.CurrentStatus("ws-1")
	if got != agent.StatusWorking {
		t.Errorf("CurrentStatus: got %s, want Working", got)
	}
}

func TestRefreshWorkspaceStatuses(t *testing.T) {
	detector := NewDetector(&mockCapturer{content: ">\n"}, 50)

	workspaces := []workspace.Workspace{
		{
			ID:        "ws-1",
			AgentType: agent.Claude,
			Status:    agent.StatusWorking,
			PaneTargets: map[string]string{
				"agent": "%0",
			},
		},
	}

	refreshed, changed := RefreshWorkspaceStatuses(context.Background(), detector, workspaces)
	if !changed {
		t.Fatal("expected statuses to change")
	}
	if refreshed[0].Status != agent.StatusIdle {
		t.Fatalf("status = %s, want Idle", refreshed[0].Status)
	}
}
