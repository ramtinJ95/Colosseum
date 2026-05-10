package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/workspace"
)

var (
	flagBaseBranch     string
	flagMode           string
	flagPrompt         string
	flagCandidateCount int
	flagAgentStrategy  string
)

func newNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new workspace",
		Args:  cobra.ExactArgs(1),
		RunE:  runNew,
	}

	cmd.Flags().StringVarP(&flagPath, "path", "p", ".", "project directory path")
	cmd.Flags().StringVarP(&flagAgent, "agent", "a", cfg.Defaults.Agent, "agent type (claude, codex, opencode, pi-agent)")
	cmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "git branch name")
	cmd.Flags().StringVar(&flagBaseBranch, "base", "", "base branch for managed worktrees")
	cmd.Flags().StringVarP(&flagLayout, "layout", "l", cfg.Defaults.Layout, "pane layout (agent, agent-shell, agent-shell-logs)")
	cmd.Flags().StringVar(&flagMode, "mode", string(workspace.CreateModeExistingCheckout), "create mode (existing-checkout, new-worktree, experiment-run)")
	cmd.Flags().StringVar(&flagPrompt, "prompt", "", "prompt to broadcast after experiment creation")
	cmd.Flags().IntVar(&flagCandidateCount, "count", 2, "number of experiment candidates when using the selected-agent strategy")
	cmd.Flags().StringVar(&flagAgentStrategy, "experiment-agents", string(workspace.ExperimentAgentAllSupported), "experiment agent strategy (all-supported, selected-agent)")

	return cmd
}

func runNew(_ *cobra.Command, args []string) error {
	result, err := createWorkspace(context.Background(), createWorkspaceOptions{
		Title:          args[0],
		Path:           flagPath,
		Agent:          flagAgent,
		Branch:         flagBranch,
		BaseBranch:     flagBaseBranch,
		Layout:         flagLayout,
		Mode:           flagMode,
		Prompt:         flagPrompt,
		CandidateCount: flagCandidateCount,
		AgentStrategy:  flagAgentStrategy,
	})
	if err != nil {
		return err
	}

	if result.Experiment != nil {
		fmt.Printf("Created experiment %q with %d workspaces\n", result.Experiment.Title, len(result.Workspaces))
		if len(result.Broadcast.Delivered) > 0 || len(result.Broadcast.Failed) > 0 {
			fmt.Println(formatBroadcastStatus(result.Broadcast))
		}
		return nil
	}
	if result.Workspace != nil {
		if result.Workspace.Branch != "" && workspace.CreateMode(flagMode) == workspace.CreateModeNewWorktree {
			fmt.Printf("Created workspace %q (session: %s, branch: %s)\n", result.Workspace.Title, result.Workspace.SessionName, result.Workspace.Branch)
			return nil
		}
		fmt.Printf("Created workspace %q (session: %s)\n", result.Workspace.Title, result.Workspace.SessionName)
	}
	return nil
}

func formatBroadcastStatus(result workspace.BroadcastResult) string {
	delivered := len(result.Delivered)
	failed := len(result.Failed)

	switch {
	case delivered > 0 && failed == 0:
		return fmt.Sprintf("Broadcast sent to %d workspace%s", delivered, pluralize(delivered))
	case delivered > 0 && failed > 0:
		return fmt.Sprintf("Broadcast sent to %d workspace%s (%d failed)", delivered, pluralize(delivered), failed)
	case failed > 0:
		return fmt.Sprintf("Broadcast failed for %d workspace%s", failed, pluralize(failed))
	default:
		return "Broadcast did not target any workspaces"
	}
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
