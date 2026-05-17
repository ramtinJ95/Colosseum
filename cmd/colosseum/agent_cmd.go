package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	agentpkg "github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/cliapi"
	statuspkg "github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type agentReportOptions struct {
	Workspace string
	Pane      string
	Agent     string
	Status    string
	Source    string
}

type agentReleaseOptions struct {
	Workspace string
	Pane      string
	Agent     string
	Source    string
}

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent integration operations",
	}
	cmd.AddCommand(newAgentReportCmd(), newAgentReleaseCmd())
	return cmd
}

func newAgentReportCmd() *cobra.Command {
	opts := agentReportOptions{Pane: "agent", Source: "manual"}
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Report authoritative agent status for a workspace pane",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentReport(cmd.Context(), cmd.OutOrStdout(), opts, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "workspace ID or exact title")
	cmd.Flags().StringVar(&opts.Pane, "pane", opts.Pane, "pane role or tmux pane target")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "reporting agent type")
	cmd.Flags().StringVar(&opts.Status, "status", "", "agent status (working, waiting, blocked, idle, stopped, error, unknown)")
	cmd.Flags().StringVar(&opts.Source, "source", opts.Source, "report source")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("workspace")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}

func newAgentReleaseCmd() *cobra.Command {
	opts := agentReleaseOptions{Pane: "agent"}
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Release hook-reported agent status for a workspace pane",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentRelease(cmd.Context(), cmd.OutOrStdout(), opts, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "workspace ID or exact title")
	cmd.Flags().StringVar(&opts.Pane, "pane", opts.Pane, "pane role or tmux pane target")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type to release")
	cmd.Flags().StringVar(&opts.Source, "source", "", "report source to release")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("workspace")
	return cmd
}

func runAgentReport(ctx context.Context, out io.Writer, opts agentReportOptions, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	return reportAgentStatus(ctx, out, store, opts, jsonOutput)
}

func runAgentRelease(ctx context.Context, out io.Writer, opts agentReleaseOptions, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	return releaseAgentStatus(ctx, out, store, opts, jsonOutput)
}

func reportAgentStatus(_ context.Context, out io.Writer, store *workspace.Store, opts agentReportOptions, jsonOutput bool) error {
	parsedStatus, err := parseAgentStatus(opts.Status)
	if err != nil {
		return err
	}

	var ws workspace.Workspace
	var report workspace.AgentStatusReport
	if err := store.UpdateState(func(state *workspace.State) error {
		resolved, err := resolveWorkspace(state.Workspaces, opts.Workspace)
		if err != nil {
			return err
		}
		pane := normalizedPane(opts.Pane)
		target, err := resolvePaneTarget(resolved, pane)
		if err != nil {
			return err
		}
		if !isAgentPaneReport(resolved, pane, target) {
			return fmt.Errorf("agent status reports only support the agent pane")
		}

		agentType := resolved.AgentType
		if strings.TrimSpace(opts.Agent) != "" {
			agentType = agentpkg.AgentType(strings.TrimSpace(opts.Agent))
			if !agentpkg.IsSupported(agentType) {
				return fmt.Errorf("unsupported agent type %q", agentType)
			}
			if agentType != resolved.AgentType {
				return fmt.Errorf("reported agent %q does not match workspace agent %q", agentType, resolved.AgentType)
			}
		}

		report = workspace.AgentStatusReport{
			WorkspaceID: resolved.ID,
			Pane:        pane,
			AgentType:   agentType,
			Status:      parsedStatus.String(),
			Source:      strings.TrimSpace(opts.Source),
			ReportedAt:  time.Now(),
		}
		state.AgentStatusReports = upsertAgentStatusReport(state.AgentStatusReports, report)
		for i := range state.Workspaces {
			if state.Workspaces[i].ID == resolved.ID {
				state.Workspaces[i].Status = parsedStatus
				ws = state.Workspaces[i]
				return nil
			}
		}
		return fmt.Errorf("workspace %q not found", resolved.ID)
	}); err != nil {
		return err
	}

	if jsonOutput {
		return writeJSON(out, cliapi.AgentReportResponse{Workspace: cliapi.NewWorkspace(ws), Report: cliapi.NewAgentStatusReport(report), Action: "report"})
	}
	_, err = fmt.Fprintf(out, "Reported %s for %s/%s\n", report.Status, ws.Title, report.Pane)
	return err
}

func releaseAgentStatus(_ context.Context, out io.Writer, store *workspace.Store, opts agentReleaseOptions, jsonOutput bool) error {
	var ws workspace.Workspace
	var released int
	if err := store.UpdateState(func(state *workspace.State) error {
		resolved, err := resolveWorkspace(state.Workspaces, opts.Workspace)
		if err != nil {
			return err
		}
		pane := normalizedPane(opts.Pane)
		target, err := resolvePaneTarget(resolved, pane)
		if err != nil {
			return err
		}
		if !isAgentPaneReport(resolved, pane, target) {
			return fmt.Errorf("agent status reports only support the agent pane")
		}

		ws = resolved
		state.AgentStatusReports, released = filterReleasedAgentStatusReports(state.AgentStatusReports, resolved, pane, opts.Agent, opts.Source)
		if released > 0 {
			ws.Status = agentpkg.StatusUnknown
			if reported, ok := statuspkg.SelectReport(resolved, "agent", state.AgentStatusReports, time.Now(), statuspkg.DefaultReportMaxAge); ok {
				ws.Status = reported
			}
			for i := range state.Workspaces {
				if state.Workspaces[i].ID == resolved.ID {
					state.Workspaces[i].Status = ws.Status
					break
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if jsonOutput {
		return writeJSON(out, cliapi.WorkspaceActionResponse{Workspace: cliapi.NewWorkspace(ws), Action: "release"})
	}
	_, err := fmt.Fprintf(out, "Released %d report(s) for %s/%s\n", released, ws.Title, normalizedPane(opts.Pane))
	return err
}

func upsertAgentStatusReport(reports []workspace.AgentStatusReport, next workspace.AgentStatusReport) []workspace.AgentStatusReport {
	for i, report := range reports {
		if report.WorkspaceID == next.WorkspaceID && report.Pane == next.Pane && report.Source == next.Source {
			reports[i] = next
			return reports
		}
	}
	return append(reports, next)
}

func filterReleasedAgentStatusReports(reports []workspace.AgentStatusReport, ws workspace.Workspace, pane string, agentValue string, source string) ([]workspace.AgentStatusReport, int) {
	agentValue = strings.TrimSpace(agentValue)
	source = strings.TrimSpace(source)
	target := ws.PaneTargets[pane]
	if target == "" && pane == ws.PaneTargets["agent"] {
		target = pane
		pane = "agent"
	}
	filtered := reports[:0]
	released := 0
	for _, report := range reports {
		matches := report.WorkspaceID == ws.ID && (report.Pane == pane || (target != "" && report.Pane == target))
		if matches && agentValue != "" {
			matches = report.AgentType == agentpkg.AgentType(agentValue)
		}
		if matches && source != "" {
			matches = report.Source == source
		}
		if matches {
			released++
			continue
		}
		filtered = append(filtered, report)
	}
	return filtered, released
}

func normalizedPane(pane string) string {
	pane = strings.TrimSpace(pane)
	if pane == "" {
		return "agent"
	}
	return pane
}

func isAgentPaneReport(ws workspace.Workspace, pane string, target string) bool {
	return pane == "agent" || (target != "" && target == ws.PaneTargets["agent"])
}
