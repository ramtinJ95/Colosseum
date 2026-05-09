package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/cliapi"
	"github.com/ramtinj/colosseum/internal/status"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Scriptable status operations",
	}
	cmd.AddCommand(newStatusGetCmd())
	return cmd
}

func newStatusGetCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "get <workspace>",
		Short: "Get workspace status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusGet(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func runStatusGet(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget string, jsonOutput bool) error {
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
	ws, err := resolveWorkspace(workspaces, workspaceTarget)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(out, cliapi.StatusResponse{Workspace: cliapi.NewWorkspace(ws), Status: ws.Status.String()})
	}
	_, err = fmt.Fprintln(out, ws.Status)
	return err
}
