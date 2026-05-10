package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type workspaceStore interface {
	List() ([]workspace.Workspace, error)
	Save([]workspace.Workspace) error
}

type workspaceStateStore interface {
	LoadState() (workspace.State, error)
}

type workspaceStatusDetector interface {
	Detect(ctx context.Context, paneTarget string, agentType agent.AgentType) (agent.Status, string, error)
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func resolveWorkspace(workspaces []workspace.Workspace, target string) (workspace.Workspace, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return workspace.Workspace{}, fmt.Errorf("workspace target is required")
	}

	for _, ws := range workspaces {
		if ws.ID == target {
			return ws, nil
		}
	}

	var matches []workspace.Workspace
	for _, ws := range workspaces {
		if ws.Title == target {
			matches = append(matches, ws)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return workspace.Workspace{}, fmt.Errorf("workspace %q not found", target)
	default:
		return workspace.Workspace{}, fmt.Errorf("workspace title %q is ambiguous; use an ID", target)
	}
}

func resolvePaneTarget(ws workspace.Workspace, pane string) (string, error) {
	pane = strings.TrimSpace(pane)
	if pane == "" {
		pane = "agent"
	}
	if target := ws.PaneTargets[pane]; target != "" {
		return target, nil
	}
	if strings.HasPrefix(pane, "%") {
		return pane, nil
	}
	return "", fmt.Errorf("workspace %q has no %q pane", ws.Title, pane)
}

func refreshWorkspaceStatuses(ctx context.Context, store workspaceStore, detector *status.Detector) ([]workspace.Workspace, error) {
	workspaces, reports, err := loadWorkspacesAndReports(store)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	workspaces, changed := status.RefreshWorkspaceStatusesWithReports(ctx, detector, workspaces, reports)
	if changed {
		if err := store.Save(workspaces); err != nil {
			return nil, fmt.Errorf("saving refreshed statuses: %w", err)
		}
	}
	return workspaces, nil
}

func loadWorkspacesAndReports(store workspaceStore) ([]workspace.Workspace, []workspace.AgentStatusReport, error) {
	if stateStore, ok := store.(workspaceStateStore); ok {
		state, err := stateStore.LoadState()
		if err != nil {
			return nil, nil, err
		}
		return state.Workspaces, state.AgentStatusReports, nil
	}
	workspaces, err := store.List()
	return workspaces, nil, err
}

func detectWorkspaceStatus(ctx context.Context, ws workspace.Workspace, detector workspaceStatusDetector) (agent.Status, error) {
	return detectWorkspaceStatusWithReports(ctx, ws, detector, nil)
}

func detectWorkspaceStatusWithReports(ctx context.Context, ws workspace.Workspace, detector workspaceStatusDetector, reports []workspace.AgentStatusReport) (agent.Status, error) {
	if reported, ok := status.SelectReport(ws, "agent", reports, time.Now(), status.DefaultReportMaxAge); ok {
		return reported, nil
	}
	agentPane, ok := ws.PaneTargets["agent"]
	if !ok || strings.TrimSpace(agentPane) == "" {
		return agent.StatusStopped, nil
	}
	detected, _, err := detector.Detect(ctx, agentPane, ws.AgentType)
	if err != nil {
		if tmux.IsPaneNotFound(err) || tmux.IsSessionNotFound(err) {
			return agent.StatusStopped, nil
		}
		return agent.StatusUnknown, fmt.Errorf("detect status for workspace %q: %w", ws.Title, err)
	}
	return detected, nil
}

func parseAgentStatus(value string) (agent.Status, error) {
	return agent.ParseStatus(value)
}
