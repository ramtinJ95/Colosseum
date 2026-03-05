package status

import (
	"context"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func RefreshWorkspaceStatuses(ctx context.Context, detector *Detector, workspaces []workspace.Workspace) ([]workspace.Workspace, bool) {
	if detector == nil {
		return workspaces, false
	}

	changed := false
	for i := range workspaces {
		next := workspaces[i].Status
		agentPane, ok := workspaces[i].PaneTargets["agent"]
		if !ok || agentPane == "" {
			next = agent.StatusStopped
		} else {
			status, _, err := detector.Detect(ctx, agentPane, workspaces[i].AgentType)
			if err != nil {
				next = agent.StatusStopped
			} else {
				next = status
			}
		}

		if next != workspaces[i].Status {
			workspaces[i].Status = next
			changed = true
		}
	}

	return workspaces, changed
}
