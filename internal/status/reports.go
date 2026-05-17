package status

import (
	"context"
	"strings"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

const DefaultReportMaxAge = 30 * time.Minute

func ResolveWorkspaceStatus(ctx context.Context, detector *Detector, ws workspace.Workspace, reports []workspace.AgentStatusReport) (agent.Status, string, error) {
	if reported, ok := SelectReport(ws, "agent", reports, time.Now(), DefaultReportMaxAge); ok {
		return reported, "", nil
	}

	agentPane, ok := ws.PaneTargets["agent"]
	if !ok || strings.TrimSpace(agentPane) == "" {
		return agent.StatusStopped, "", nil
	}
	return detector.Detect(ctx, agentPane, ws.AgentType)
}

func SelectReport(ws workspace.Workspace, pane string, reports []workspace.AgentStatusReport, now time.Time, maxAge time.Duration) (agent.Status, bool) {
	pane = strings.TrimSpace(pane)
	if pane == "" {
		pane = "agent"
	}
	target := ws.PaneTargets[pane]

	var newest workspace.AgentStatusReport
	found := false
	for _, report := range reports {
		if report.WorkspaceID != ws.ID {
			continue
		}
		if report.AgentType != "" && report.AgentType != ws.AgentType {
			continue
		}
		if !reportMatchesPane(report.Pane, pane, target) {
			continue
		}
		if maxAge > 0 && !report.ReportedAt.IsZero() && now.Sub(report.ReportedAt) > maxAge {
			continue
		}
		if !found || report.ReportedAt.After(newest.ReportedAt) {
			newest = report
			found = true
		}
	}
	if !found {
		return agent.StatusUnknown, false
	}

	parsed, err := agent.ParseStatus(newest.Status)
	if err != nil {
		return agent.StatusUnknown, false
	}
	return parsed, true
}

func reportMatchesPane(reportPane string, pane string, target string) bool {
	reportPane = strings.TrimSpace(reportPane)
	return reportPane == pane || (target != "" && reportPane == target)
}
