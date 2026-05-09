package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type fakeWorkspaceStore struct {
	workspaces []workspace.Workspace
	saved      [][]workspace.Workspace
}

func (f *fakeWorkspaceStore) List() ([]workspace.Workspace, error) {
	result := make([]workspace.Workspace, len(f.workspaces))
	copy(result, f.workspaces)
	return result, nil
}

func (f *fakeWorkspaceStore) Save(workspaces []workspace.Workspace) error {
	result := make([]workspace.Workspace, len(workspaces))
	copy(result, workspaces)
	f.saved = append(f.saved, result)
	return nil
}

type fakePaneClient struct {
	content      string
	captureCalls []fakeCaptureCall
	panes        []tmux.PaneInfo
}

type fakeCaptureCall struct {
	target string
	lines  int
}

func (f *fakePaneClient) CapturePane(_ context.Context, target string, lines int) (string, error) {
	f.captureCalls = append(f.captureCalls, fakeCaptureCall{target: target, lines: lines})
	return f.content, nil
}

func (f *fakePaneClient) ListPanes(_ context.Context, _ string) ([]tmux.PaneInfo, error) {
	return f.panes, nil
}

type fakeStatusDetector struct {
	statuses []agent.Status
	err      error
	calls    int
}

func (f *fakeStatusDetector) Detect(_ context.Context, _ string, _ agent.AgentType) (agent.Status, string, error) {
	f.calls++
	if f.err != nil {
		return agent.StatusUnknown, "", f.err
	}
	status := f.statuses[len(f.statuses)-1]
	if f.calls <= len(f.statuses) {
		status = f.statuses[f.calls-1]
	}
	return status, "", nil
}

func TestResolveWorkspacePrefersIDOverTitle(t *testing.T) {
	workspaces := []workspace.Workspace{
		{ID: "target", Title: "other"},
		{ID: "other", Title: "target"},
	}

	got, err := resolveWorkspace(workspaces, "target")
	if err != nil {
		t.Fatalf("resolveWorkspace: %v", err)
	}
	if got.ID != "target" {
		t.Fatalf("resolved ID = %q, want target", got.ID)
	}
}

func TestResolveWorkspaceRejectsAmbiguousTitle(t *testing.T) {
	workspaces := []workspace.Workspace{
		{ID: "ws-1", Title: "same"},
		{ID: "ws-2", Title: "same"},
	}

	_, err := resolveWorkspace(workspaces, "same")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("resolveWorkspace error = %v, want ambiguous", err)
	}
}

func TestRunPaneReadWithDepsJSON(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	paneClient := &fakePaneClient{content: "hello\nworld"}
	var out bytes.Buffer

	if err := runPaneReadWithDeps(context.Background(), &out, store, paneClient, "alpha", "agent", 42, true); err != nil {
		t.Fatalf("runPaneReadWithDeps: %v", err)
	}
	if len(paneClient.captureCalls) != 1 || paneClient.captureCalls[0].target != "%3" || paneClient.captureCalls[0].lines != 42 {
		t.Fatalf("captureCalls = %+v, want target %%3 lines 42", paneClient.captureCalls)
	}

	var got struct {
		Workspace struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"workspace"`
		Target  string `json:"target"`
		Lines   int    `json:"lines"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, out.String())
	}
	if got.Workspace.ID != "ws-1" || got.Workspace.Status != "Unknown" || got.Target != "%3" || got.Lines != 42 || got.Content != "hello\nworld" {
		t.Fatalf("JSON response = %+v", got)
	}
}

func TestWaitForWorkspaceStatusReturnsAfterMatch(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	detector := &fakeStatusDetector{statuses: []agent.Status{agent.StatusWorking, agent.StatusIdle}}

	ws, got, _, err := waitForWorkspaceStatus(context.Background(), store, detector, "ws-1", agent.StatusIdle, time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("waitForWorkspaceStatus: %v", err)
	}
	if ws.ID != "ws-1" || got != agent.StatusIdle {
		t.Fatalf("got workspace %q status %s", ws.ID, got)
	}
	if detector.calls != 2 {
		t.Fatalf("detector calls = %d, want 2", detector.calls)
	}
}

func TestWaitForWorkspaceStatusTimesOut(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	detector := &fakeStatusDetector{statuses: []agent.Status{agent.StatusWorking}}

	_, got, _, err := waitForWorkspaceStatus(context.Background(), store, detector, "alpha", agent.StatusIdle, 5*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if got != agent.StatusWorking {
		t.Fatalf("last status = %s, want Working", got)
	}
}

func TestWaitForWorkspaceStatusPropagatesUnexpectedDetectionError(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	detector := &fakeStatusDetector{err: errors.New("tmux unavailable")}

	_, got, _, err := waitForWorkspaceStatus(context.Background(), store, detector, "alpha", agent.StatusStopped, time.Second, time.Millisecond)
	if err == nil {
		t.Fatal("expected detection error")
	}
	if !strings.Contains(err.Error(), "detect status for workspace") || !strings.Contains(err.Error(), "tmux unavailable") {
		t.Fatalf("error = %v, want contextual detection error", err)
	}
	if got != agent.StatusUnknown {
		t.Fatalf("status = %s, want Unknown", got)
	}
}

func TestWaitForWorkspaceStatusTreatsMissingPaneAsStopped(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	detector := &fakeStatusDetector{err: &tmux.TmuxError{Args: []string{"capture-pane"}, Stderr: "can't find pane: %3"}}

	ws, got, _, err := waitForWorkspaceStatus(context.Background(), store, detector, "alpha", agent.StatusStopped, time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("waitForWorkspaceStatus: %v", err)
	}
	if ws.ID != "ws-1" || got != agent.StatusStopped {
		t.Fatalf("got workspace %q status %s, want ws-1 Stopped", ws.ID, got)
	}
}

func TestWaitForPaneOutputTimesOut(t *testing.T) {
	store := &fakeWorkspaceStore{workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		PaneTargets: map[string]string{"agent": "%3"},
	}}}
	paneClient := &fakePaneClient{content: "not yet"}

	_, _, _, err := waitForPaneOutput(context.Background(), store, paneClient, "alpha", "agent", "done", 10, 5*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
	if len(paneClient.captureCalls) == 0 {
		t.Fatal("expected pane capture calls")
	}
}
