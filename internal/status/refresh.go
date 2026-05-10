package status

import (
	"context"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func RefreshWorkspaceStatuses(ctx context.Context, detector *Detector, workspaces []workspace.Workspace) ([]workspace.Workspace, bool) {
	return RefreshWorkspaceStatusesWithReports(ctx, detector, workspaces, nil)
}

func RefreshWorkspaceStatusesWithReports(ctx context.Context, detector *Detector, workspaces []workspace.Workspace, reports []workspace.AgentStatusReport) ([]workspace.Workspace, bool) {
	if detector == nil {
		return workspaces, false
	}

	changed := false
	for i := range workspaces {
		next, _, err := ResolveWorkspaceStatus(ctx, detector, workspaces[i], reports)
		if err != nil {
			next = agent.StatusStopped
		}

		if next != workspaces[i].Status {
			workspaces[i].Status = next
			changed = true
		}
	}

	return workspaces, changed
}
