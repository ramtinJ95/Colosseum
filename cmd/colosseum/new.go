package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
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
	cmd.Flags().StringVarP(&flagLayout, "layout", "l", cfg.Defaults.Layout, "pane layout (agent, agent-shell, agent-shell-logs)")

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

	ws, err := mgr.Create(context.Background(), name, agentType, absPath, flagBranch, layout)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	fmt.Printf("Created workspace %q (session: %s)\n", ws.Title, ws.SessionName)
	return nil
}
