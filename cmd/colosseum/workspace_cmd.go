package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/cliapi"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Scriptable workspace operations",
	}
	cmd.AddCommand(
		newWorkspaceListCmd(),
		newWorkspaceGetCmd(),
		newWorkspaceCreateCmd(),
		newWorkspaceFocusCmd(),
		newWorkspaceDeleteCmd(),
	)
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

func newWorkspaceCreateCmd() *cobra.Command {
	opts := defaultCreateWorkspaceOptions("")
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workspace or experiment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWorkspaceCreate(cmd.Context(), cmd.OutOrStdout(), opts, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&opts.Title, "title", "", "workspace or experiment title")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", opts.Path, "project directory path")
	cmd.Flags().StringVarP(&opts.Agent, "agent", "a", opts.Agent, "agent type (claude, codex, opencode, pi-agent)")
	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "git branch name")
	cmd.Flags().StringVar(&opts.BaseBranch, "base", "", "base branch for managed worktrees")
	cmd.Flags().StringVarP(&opts.Layout, "layout", "l", opts.Layout, "pane layout (agent, agent-shell, agent-shell-logs)")
	cmd.Flags().StringVar(&opts.Mode, "mode", opts.Mode, "create mode (existing-checkout, new-worktree, experiment-run)")
	cmd.Flags().StringVar(&opts.Prompt, "prompt", "", "prompt to broadcast after experiment creation")
	cmd.Flags().IntVar(&opts.CandidateCount, "count", opts.CandidateCount, "number of experiment candidates when using the selected-agent strategy")
	cmd.Flags().StringVar(&opts.AgentStrategy, "experiment-agents", opts.AgentStrategy, "experiment agent strategy (all-supported, selected-agent)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newWorkspaceFocusCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "focus <id-or-title>",
		Short: "Focus a workspace tmux session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceFocus(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func newWorkspaceDeleteCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "delete <id-or-title>",
		Short: "Delete a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceDelete(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func runWorkspaceCreate(ctx context.Context, out io.Writer, opts createWorkspaceOptions, jsonOutput bool) error {
	result, err := createWorkspace(ctx, opts)
	if err != nil {
		return err
	}
	if jsonOutput {
		workspaces := cliapi.NewWorkspaces(result.Workspaces)
		if result.Workspace != nil {
			workspaces = nil
		}
		return writeJSON(out, cliapi.WorkspaceCreateResponse{
			Workspace:  cliapiWorkspacePtr(result.Workspace),
			Workspaces: workspaces,
			Experiment: result.Experiment,
			Broadcast:  cliapi.NewBroadcastResult(result.Broadcast),
		})
	}
	if result.Experiment != nil {
		if _, err := fmt.Fprintf(out, "Created experiment %q with %d workspaces\n", result.Experiment.Title, len(result.Workspaces)); err != nil {
			return err
		}
		if len(result.Broadcast.Delivered) > 0 || len(result.Broadcast.Failed) > 0 {
			_, err := fmt.Fprintln(out, formatBroadcastStatus(result.Broadcast))
			return err
		}
		return nil
	}
	if result.Workspace != nil {
		if result.Workspace.Branch != "" && workspace.CreateMode(opts.Mode) == workspace.CreateModeNewWorktree {
			_, err := fmt.Fprintf(out, "Created workspace %q (session: %s, branch: %s)\n", result.Workspace.Title, result.Workspace.SessionName, result.Workspace.Branch)
			return err
		}
		_, err := fmt.Fprintf(out, "Created workspace %q (session: %s)\n", result.Workspace.Title, result.Workspace.SessionName)
		return err
	}
	return nil
}

func cliapiWorkspacePtr(ws *workspace.Workspace) *cliapi.Workspace {
	if ws == nil {
		return nil
	}
	result := cliapi.NewWorkspace(*ws)
	return &result
}

func runWorkspaceFocus(ctx context.Context, out io.Writer, workspaceTarget string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	mgr := newManager(store, client)
	ws, err := resolveWorkspaceFromManager(mgr, workspaceTarget)
	if err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		if err := client.AttachSession(ctx, workspaceSessionName(ws)); err != nil {
			return err
		}
	} else if err := mgr.SwitchTo(ctx, ws.ID); err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(out, cliapi.WorkspaceActionResponse{Workspace: cliapi.NewWorkspace(ws), Action: "focus"})
	}
	_, err = fmt.Fprintf(out, "Focused workspace %q\n", ws.Title)
	return err
}

func runWorkspaceDelete(ctx context.Context, out io.Writer, workspaceTarget string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	mgr := newManager(store, client)
	ws, err := resolveWorkspaceFromManager(mgr, workspaceTarget)
	if err != nil {
		return err
	}
	if err := mgr.Delete(ctx, ws.ID); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if jsonOutput {
		return writeJSON(out, cliapi.WorkspaceActionResponse{Workspace: cliapi.NewWorkspace(ws), Action: "delete"})
	}
	_, err = fmt.Fprintf(out, "Deleted workspace %q\n", ws.Title)
	return err
}

func resolveWorkspaceFromManager(mgr *workspace.Manager, target string) (workspace.Workspace, error) {
	workspaces, err := mgr.List()
	if err != nil {
		return workspace.Workspace{}, fmt.Errorf("list workspaces: %w", err)
	}
	return resolveWorkspace(workspaces, target)
}
