package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/cliapi"
	"github.com/ramtinj/colosseum/internal/status"
)

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Scriptable workspace operations",
	}
	cmd.AddCommand(newWorkspaceListCmd(), newWorkspaceGetCmd())
	return cmd
}

func newWorkspaceListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWorkspaceList(cmd.Context(), cmd.OutOrStdout(), jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func newWorkspaceGetCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "get <id-or-title>",
		Short: "Get a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceGet(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func runWorkspaceList(ctx context.Context, out interface{ Write([]byte) (int, error) }, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	workspaces, err := refreshWorkspaceStatuses(ctx, store, detector)
	if err != nil {
		return err
	}

	if jsonOutput {
		return writeJSON(out, cliapi.WorkspaceListResponse{Workspaces: cliapi.NewWorkspaces(workspaces)})
	}
	if len(workspaces) == 0 {
		_, err := fmt.Fprintln(out, "No workspaces. Create one with: colosseum new <name>")
		return err
	}
	for _, ws := range workspaces {
		branch := ""
		if ws.Branch != "" {
			branch = fmt.Sprintf(" [%s]", ws.Branch)
		}
		if _, err := fmt.Fprintf(out, "  %s %s%s (%s · %s)\n", statusIcon(ws.Status), ws.Title, branch, ws.AgentType, ws.Status); err != nil {
			return err
		}
	}
	return nil
}

func runWorkspaceGet(ctx context.Context, out interface{ Write([]byte) (int, error) }, target string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	workspaces, err := refreshWorkspaceStatuses(ctx, store, detector)
	if err != nil {
		return err
	}
	ws, err := resolveWorkspace(workspaces, target)
	if err != nil {
		return err
	}

	if jsonOutput {
		return writeJSON(out, cliapi.WorkspaceResponse{Workspace: cliapi.NewWorkspace(ws)})
	}
	_, err = fmt.Fprintf(out, "%s %s (%s · %s)\n", statusIcon(ws.Status), ws.Title, ws.AgentType, ws.Status)
	return err
}
