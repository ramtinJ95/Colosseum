package status

import (
	"context"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestPollerDetectsStatusChange(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent)
	h.start()

	update := h.nextUpdate()
	if update.WorkspaceID != "ws-1" {
		t.Errorf("workspace ID = %q, want ws-1", update.WorkspaceID)
	}
	if update.Current != agent.StatusWorking {
		t.Errorf("status = %s, want Working", update.Current)
	}
	if update.Previous != agent.StatusUnknown {
		t.Errorf("previous = %s, want Unknown", update.Previous)
	}
}

func TestPollerNoUpdateWhenStatusUnchanged(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent)
	h.start()

	_ = h.nextUpdate()
	h.expectNoUpdate(200 * time.Millisecond)
}

func TestPollerStatusTransition(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent)
	h.start()

	update := h.nextUpdate()
	if update.Current != agent.StatusWorking {
		t.Fatalf("first update = %s, want Working", update.Current)
	}

	h.setContent(testIdleContent)
	update = h.nextUpdate()
	if update.Previous != agent.StatusWorking {
		t.Errorf("previous = %s, want Working", update.Previous)
	}
	if update.Current != agent.StatusIdle {
		t.Errorf("current = %s, want Idle", update.Current)
	}
}

func TestPollerCurrentStatus(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent)
	h.start()

	_ = h.nextUpdate()
	if got := h.poller.CurrentStatus("ws-1"); got != agent.StatusWorking {
		t.Errorf("CurrentStatus = %s, want Working", got)
	}
}

func TestPollerSpikeWindowDelaysNonUrgentTransition(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent, WithSpikeWindow(200*time.Millisecond))
	h.start()

	update := h.nextUpdate()
	if update.Current != agent.StatusWorking {
		t.Fatalf("initial status = %s, want Working", update.Current)
	}

	h.setContent(testIdleContent)
	transitionTime := time.Now()
	update = h.nextUpdate()
	if update.Current != agent.StatusIdle {
		t.Fatalf("transition status = %s, want Idle", update.Current)
	}
	if elapsed := time.Since(transitionTime); elapsed < 150*time.Millisecond {
		t.Errorf("spike window should delay transition, took only %v", elapsed)
	}
}

func TestPollerUrgentStatusBypassesSpikeWindow(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testWorkingContent, WithSpikeWindow(5*time.Second))
	h.start()

	_ = h.nextUpdate()
	h.setContent(testWaitingContent)

	update := h.nextUpdate()
	if update.Current != agent.StatusWaiting {
		t.Fatalf("urgent transition = %s, want Waiting", update.Current)
	}
}

func TestPollerTitleUpgradesUnknownToWorking(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, "some output text\n")
	h.setTitle("⠹ claude")
	h.start()

	update := h.nextUpdate()
	if update.Current != agent.StatusWorking {
		t.Errorf("title braille should upgrade Unknown to Working, got %s", update.Current)
	}
}

func TestPollerTitleDoesNotUpgradeIdle(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testIdleContent)
	h.setTitle("⠹ claude")
	h.start()

	update := h.nextUpdate()
	if update.Current != agent.StatusIdle {
		t.Errorf("title should not override definitive Idle from content, got %s", update.Current)
	}
}

func TestPollerPrefersReportedStatus(t *testing.T) {
	h := newPollerHarness(t, agent.Claude, testIdleContent)
	h.provider.reports = []workspace.AgentStatusReport{{
		WorkspaceID: "ws-1",
		Pane:        "agent",
		AgentType:   agent.Claude,
		Status:      agent.StatusWorking.String(),
		Source:      "test-hook",
		ReportedAt:  time.Now(),
	}}
	h.start()

	update := h.nextUpdate()
	if update.Current != agent.StatusWorking {
		t.Errorf("reported status should override terminal content, got %s", update.Current)
	}
}

func TestRefreshWorkspaceStatuses(t *testing.T) {
	detector := NewDetector(&mockCapturer{content: testIdleContent}, testDetectLines)
	workspaces := []workspace.Workspace{statusTestWorkspace("ws-1", agent.Claude)}
	workspaces[0].Status = agent.StatusWorking

	refreshed, changed := RefreshWorkspaceStatuses(context.Background(), detector, workspaces)
	if !changed {
		t.Fatal("expected statuses to change")
	}
	if refreshed[0].Status != agent.StatusIdle {
		t.Fatalf("status = %s, want Idle", refreshed[0].Status)
	}
}

func TestRefreshWorkspaceStatusesPrefersReport(t *testing.T) {
	detector := NewDetector(&mockCapturer{content: testIdleContent}, testDetectLines)
	workspaces := []workspace.Workspace{statusTestWorkspace("ws-1", agent.Claude)}
	reports := []workspace.AgentStatusReport{{
		WorkspaceID: "ws-1",
		Pane:        "agent",
		AgentType:   agent.Claude,
		Status:      agent.StatusWorking.String(),
		Source:      "test-hook",
		ReportedAt:  time.Now(),
	}}

	refreshed, changed := RefreshWorkspaceStatusesWithReports(context.Background(), detector, workspaces, reports)
	if !changed {
		if refreshed[0].Status != agent.StatusWorking {
			t.Fatalf("expected report to set Working")
		}
		t.Fatal("expected statuses to change")
	}
	if refreshed[0].Status != agent.StatusWorking {
		t.Fatalf("status = %s, want Working", refreshed[0].Status)
	}
}

func TestRefreshWorkspaceStatusesIgnoresStaleReport(t *testing.T) {
	detector := NewDetector(&mockCapturer{content: testIdleContent}, testDetectLines)
	workspaces := []workspace.Workspace{statusTestWorkspace("ws-1", agent.Claude)}
	reports := []workspace.AgentStatusReport{{
		WorkspaceID: "ws-1",
		Pane:        "agent",
		AgentType:   agent.Claude,
		Status:      agent.StatusWorking.String(),
		Source:      "test-hook",
		ReportedAt:  time.Now().Add(-DefaultReportMaxAge - time.Minute),
	}}

	refreshed, changed := RefreshWorkspaceStatusesWithReports(context.Background(), detector, workspaces, reports)
	if !changed {
		t.Fatal("expected fallback detection to change status")
	}
	if refreshed[0].Status != agent.StatusIdle {
		t.Fatalf("status = %s, want Idle fallback", refreshed[0].Status)
	}
}
