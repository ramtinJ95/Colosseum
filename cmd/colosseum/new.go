package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/agent"
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
	cmd.Flags().StringVarP(&flagAgent, "agent", "a", cfg.Defaults.Agent, "agent type (claude, codex, opencode)")
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
	name := args[0]
	agentType := agent.AgentType(flagAgent)

	if !agent.IsSupported(agentType) {
		return fmt.Errorf("unsupported agent type: %s (supported: %v)", flagAgent, agent.Supported())
	}

	if _, ok := agent.Get(agentType); !ok {
		return fmt.Errorf("agent type %q is not registered", flagAgent)
	}

	layout := workspace.LayoutType(flagLayout)
	if !workspace.IsValidLayout(layout) {
		return fmt.Errorf("invalid layout %q (valid: %v)", flagLayout, workspace.ValidLayouts())
	}

	absPath, err := filepath.Abs(flagPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	store := newStore()
	client := newTmuxClient()
	mgr := newManager(store, client)
	mode := workspace.CreateMode(flagMode)

	switch mode {
	case workspace.CreateModeNewWorktree:
		ws, err := mgr.CreateWithWorktree(context.Background(), workspace.ManagedWorkspaceRequest{
			Title:      name,
			AgentType:  agentType,
			RepoRoot:   absPath,
			Branch:     flagBranch,
			BaseBranch: flagBaseBranch,
			Layout:     layout,
		})
		if err != nil {
			return fmt.Errorf("create worktree workspace: %w", err)
		}
		fmt.Printf("Created workspace %q (session: %s, branch: %s)\n", ws.Title, ws.SessionName, ws.Branch)
		return nil
	case workspace.CreateModeExperimentRun:
		result, err := mgr.CreateExperiment(context.Background(), workspace.ExperimentRequest{
			Title:          name,
			Prompt:         flagPrompt,
			RepoRoot:       absPath,
			BaseBranch:     flagBaseBranch,
			CandidateCount: flagCandidateCount,
			AgentStrategy:  workspace.ExperimentAgentStrategy(flagAgentStrategy),
			AgentType:      agentType,
			Layout:         layout,
		})
		if err != nil {
			return fmt.Errorf("create experiment: %w", err)
		}
		fmt.Printf("Created experiment %q with %d workspaces\n", result.Experiment.Title, len(result.Workspaces))
		if len(result.Broadcast.Delivered) > 0 || len(result.Broadcast.Failed) > 0 {
			fmt.Println(formatBroadcastStatus(result.Broadcast))
		}
		return nil
	case workspace.CreateModeExistingCheckout:
		ws, err := mgr.CreateStandalone(context.Background(), workspace.StandaloneWorkspaceRequest{
			Title:        name,
			AgentType:    agentType,
			CheckoutPath: absPath,
			Layout:       layout,
		})
		if err != nil {
			return fmt.Errorf("create workspace: %w", err)
		}
		fmt.Printf("Created workspace %q (session: %s)\n", ws.Title, ws.SessionName)
		return nil
	default:
		return fmt.Errorf("unsupported create mode %q", flagMode)
	}
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
