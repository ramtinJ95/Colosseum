package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/status"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		RunE:  runList,
	}
}

func runList(_ *cobra.Command, _ []string) error {
	store := newStore()
	workspaces, err := store.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	client := newTmuxClient()
	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	workspaces, changed := status.RefreshWorkspaceStatuses(context.Background(), detector, workspaces)
	if changed {
		if err := store.Save(workspaces); err != nil {
			return fmt.Errorf("saving refreshed statuses: %w", err)
		}
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces. Create one with: colosseum new <name>")
		return nil
	}

	for _, ws := range workspaces {
		branch := ""
		if ws.Branch != "" {
			branch = fmt.Sprintf(" [%s]", ws.Branch)
		}
		fmt.Printf("  %s %s%s (%s · %s)\n",
			statusIcon(ws.Status), ws.Title, branch,
			ws.AgentType, ws.Status)
	}
	return nil
}

func statusIcon(s agent.Status) string {
	switch s {
	case agent.StatusWorking:
		return "●"
	case agent.StatusWaiting:
		return "◉"
	case agent.StatusIdle:
		return "○"
	case agent.StatusStopped:
		return "■"
	case agent.StatusError:
		return "✗"
	default:
		return "?"
	}
}
