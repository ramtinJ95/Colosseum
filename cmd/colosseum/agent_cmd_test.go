package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestReportAgentStatusPersistsReportAndUpdatesWorkspace(t *testing.T) {
	store := newAgentCommandTestStore(t)
	var out bytes.Buffer

	err := reportAgentStatus(context.Background(), &out, store, agentReportOptions{
		Workspace: "alpha",
		Pane:      "agent",
		Agent:     string(agent.Claude),
		Status:    "blocked",
		Source:    "claude-hook",
	}, true)
	if err != nil {
		t.Fatalf("reportAgentStatus: %v", err)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got := state.Workspaces[0].Status; got != agent.StatusWaiting {
		t.Fatalf("workspace status = %s, want Waiting", got)
	}
	if len(state.AgentStatusReports) != 1 {
		t.Fatalf("reports = %d, want 1", len(state.AgentStatusReports))
	}
	report := state.AgentStatusReports[0]
	if report.WorkspaceID != "ws-1" || report.Pane != "agent" || report.Status != agent.StatusWaiting.String() || report.Source != "claude-hook" {
		t.Fatalf("report = %+v, want ws-1 agent Waiting claude-hook", report)
	}

	var response struct {
		Action string `json:"action"`
		Report struct {
			Status string `json:"status"`
		} `json:"report"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, out.String())
	}
	if response.Action != "report" || response.Report.Status != agent.StatusWaiting.String() {
		t.Fatalf("response = %+v", response)
	}
}

func TestReleaseAgentStatusRemovesMatchingReports(t *testing.T) {
	store := newAgentCommandTestStore(t)
	if err := reportAgentStatus(context.Background(), &bytes.Buffer{}, store, agentReportOptions{Workspace: "ws-1", Pane: "agent", Status: "working", Source: "pi-hook"}, false); err != nil {
		t.Fatalf("reportAgentStatus: %v", err)
	}

	var out bytes.Buffer
	if err := releaseAgentStatus(context.Background(), &out, store, agentReleaseOptions{Workspace: "ws-1", Pane: "agent", Source: "pi-hook"}, false); err != nil {
		t.Fatalf("releaseAgentStatus: %v", err)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(state.AgentStatusReports) != 0 {
		t.Fatalf("reports = %+v, want none", state.AgentStatusReports)
	}
}

func newAgentCommandTestStore(t *testing.T) *workspace.Store {
	t.Helper()
	store := workspace.NewStore(filepath.Join(t.TempDir(), "workspaces.json"))
	if err := store.SaveState(workspace.State{Workspaces: []workspace.Workspace{{
		ID:          "ws-1",
		Title:       "alpha",
		AgentType:   agent.Claude,
		Status:      agent.StatusIdle,
		PaneTargets: map[string]string{"agent": "%3"},
	}}}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	return store
}
