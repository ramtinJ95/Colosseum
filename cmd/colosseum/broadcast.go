package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/workspace"
)

var (
	flagBroadcastPrompt     string
	flagBroadcastWorkspaces string
)

func newBroadcastCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "broadcast",
		Short: "Broadcast a prompt to multiple workspaces",
		RunE:  runBroadcast,
	}

	cmd.Flags().StringVarP(&flagBroadcastPrompt, "prompt", "p", "", "prompt text to send")
	cmd.Flags().StringVarP(&flagBroadcastWorkspaces, "workspaces", "w", "", "comma-separated workspace names")
	cmd.MarkFlagRequired("prompt")
	cmd.MarkFlagRequired("workspaces")

	return cmd
}

func runBroadcast(_ *cobra.Command, _ []string) error {
	targetNames := parseBroadcastWorkspaceNames(flagBroadcastWorkspaces)
	if len(targetNames) == 0 {
		return fmt.Errorf("broadcast requires at least one workspace name")
	}

	store := newStore()
	client := newTmuxClient()
	mgr := newManager(store, client)

	workspaces, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	workspaceIDs, err := resolveBroadcastWorkspaceIDs(workspaces, targetNames)
	if err != nil {
		return err
	}

	result, err := mgr.Broadcast(context.Background(), flagBroadcastPrompt, workspaceIDs)
	if err != nil {
		return fmt.Errorf("broadcast prompt: %w", err)
	}

	fmt.Println(broadcastStatusLine(result))
	return nil
}

func parseBroadcastWorkspaceNames(raw string) []string {
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func resolveBroadcastWorkspaceIDs(workspaces []workspace.Workspace, targetNames []string) ([]string, error) {
	byTitle := make(map[string]string, len(workspaces))
	for _, ws := range workspaces {
		byTitle[ws.Title] = ws.ID
	}

	ids := make([]string, 0, len(targetNames))
	for _, name := range targetNames {
		id, ok := byTitle[name]
		if !ok {
			return nil, fmt.Errorf("workspace %q not found", name)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func broadcastStatusLine(result workspace.BroadcastResult) string {
	delivered := len(result.Delivered)
	failed := len(result.Failed)

	switch {
	case delivered > 0 && failed == 0:
		return fmt.Sprintf("Broadcast sent to %d workspace%s", delivered, broadcastPluralSuffix(delivered))
	case delivered > 0 && failed > 0:
		return fmt.Sprintf("Broadcast sent to %d workspace%s (%d failed)", delivered, broadcastPluralSuffix(delivered), failed)
	case failed > 0:
		return fmt.Sprintf("Broadcast failed for %d workspace%s", failed, broadcastPluralSuffix(failed))
	default:
		return "Broadcast did not target any workspaces"
	}
}

func broadcastPluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
