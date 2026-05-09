package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type workspaceStore interface {
	List() ([]workspace.Workspace, error)
	Save([]workspace.Workspace) error
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
	workspaces, err := store.List()
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	workspaces, changed := status.RefreshWorkspaceStatuses(ctx, detector, workspaces)
	if changed {
		if err := store.Save(workspaces); err != nil {
			return nil, fmt.Errorf("saving refreshed statuses: %w", err)
		}
	}
	return workspaces, nil
}

func detectWorkspaceStatus(ctx context.Context, ws workspace.Workspace, detector workspaceStatusDetector) (agent.Status, error) {
	agentPane, ok := ws.PaneTargets["agent"]
	if !ok || strings.TrimSpace(agentPane) == "" {
		return ws.Status, nil
	}
	detected, _, err := detector.Detect(ctx, agentPane, ws.AgentType)
	if err != nil {
		return agent.StatusStopped, nil
	}
	return detected, nil
}

func parseAgentStatus(value string) (agent.Status, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "unknown":
		return agent.StatusUnknown, nil
	case "working":
		return agent.StatusWorking, nil
	case "waiting", "blocked":
		return agent.StatusWaiting, nil
	case "idle":
		return agent.StatusIdle, nil
	case "stopped":
		return agent.StatusStopped, nil
	case "error":
		return agent.StatusError, nil
	default:
		return agent.StatusUnknown, fmt.Errorf("unknown status %q", value)
	}
}
