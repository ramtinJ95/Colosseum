package status

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type mockCapturer struct {
	mu      sync.Mutex
	content string
	title   string
	err     error
}

func (m *mockCapturer) SetContent(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = content
}

func (m *mockCapturer) CapturePane(_ context.Context, _ string, _ int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.content, m.err
}

func (m *mockCapturer) CapturePaneTitle(_ context.Context, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.title, nil
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
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

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
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

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
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go poller.Run(ctx)

	// First update: Unknown -> Working
	update := <-poller.Updates()
	if update.Current != agent.StatusWorking {
		t.Fatalf("first update: got %s, want Working", update.Current)
	}

	// Change capturer content to idle
	capturer.SetContent(">\n")

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
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	<-poller.Updates()

	got := poller.CurrentStatus("ws-1")
	if got != agent.StatusWorking {
		t.Errorf("CurrentStatus: got %s, want Working", got)
	}
}

func TestPollerSpikeWindowDelaysNonUrgentTransition(t *testing.T) {
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
	// Short spike window so the test doesn't take long, but enough to require multiple polls.
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(200*time.Millisecond), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go poller.Run(ctx)

	// Initial detection (Unknown → Working) is immediate.
	update := <-poller.Updates()
	if update.Current != agent.StatusWorking {
		t.Fatalf("initial: got %s, want Working", update.Current)
	}

	// Switch to Idle — non-urgent, should be delayed by spike window.
	capturer.SetContent(">\n")
	transitionTime := time.Now()

	update = <-poller.Updates()
	elapsed := time.Since(transitionTime)
	if update.Current != agent.StatusIdle {
		t.Fatalf("transition: got %s, want Idle", update.Current)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("spike window should delay transition, but it took only %v", elapsed)
	}
}

func TestPollerUrgentStatusBypassesSpikeWindow(t *testing.T) {
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
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(5*time.Second), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go poller.Run(ctx)

	// Initial detection.
	<-poller.Updates()

	// Switch to Waiting (urgent) — should bypass the 5s spike window.
	capturer.SetContent("Do you want to allow this?\n\n>")

	update := <-poller.Updates()
	if update.Current != agent.StatusWaiting {
		t.Fatalf("urgent transition: got %s, want Waiting", update.Current)
	}
}

func TestPollerTitleUpgradesUnknownToWorking(t *testing.T) {
	// Content that produces StatusUnknown (no pattern matches).
	// Title has braille spinner → should upgrade to Working.
	capturer := &mockCapturer{
		content: "some output text\n",
		title:   "⠹ claude",
	}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	update := <-poller.Updates()
	if update.Current != agent.StatusWorking {
		t.Errorf("title braille should upgrade Unknown to Working, got %s", update.Current)
	}
}

func TestPollerTitleDoesNotUpgradeIdle(t *testing.T) {
	// Content definitively says Idle (prompt visible).
	// Title has braille spinner → should NOT upgrade (title may be stale).
	capturer := &mockCapturer{
		content: ">\n",
		title:   "⠹ claude",
	}
	detector := NewDetector(capturer, 50)

	ws := workspace.Workspace{
		ID:        "ws-1",
		AgentType: agent.Claude,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
	provider := &mockProvider{workspaces: []workspace.Workspace{ws}}
	poller := NewPoller(detector, provider, 50*time.Millisecond, WithSpikeWindow(0), WithHysteresisWindow(0))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go poller.Run(ctx)

	update := <-poller.Updates()
	if update.Current != agent.StatusIdle {
		t.Errorf("title should not override definitive Idle from content, got %s", update.Current)
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
