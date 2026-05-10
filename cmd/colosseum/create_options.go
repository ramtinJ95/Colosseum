package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type createWorkspaceOptions struct {
	Title          string
	Path           string
	Agent          string
	Branch         string
	BaseBranch     string
	Layout         string
	Mode           string
	Prompt         string
	CandidateCount int
	AgentStrategy  string
}

type createWorkspaceResult struct {
	Workspace  *workspace.Workspace
	Experiment *workspace.Experiment
	Workspaces []workspace.Workspace
	Broadcast  workspace.BroadcastResult
}

func createWorkspace(ctx context.Context, opts createWorkspaceOptions) (createWorkspaceResult, error) {
	agentType := agent.AgentType(opts.Agent)
	if !agent.IsSupported(agentType) {
		return createWorkspaceResult{}, fmt.Errorf("unsupported agent type: %s (supported: %v)", opts.Agent, agent.Supported())
	}
	if _, ok := agent.Get(agentType); !ok {
		return createWorkspaceResult{}, fmt.Errorf("agent type %q is not registered", opts.Agent)
	}

	layout := workspace.LayoutType(opts.Layout)
	if !workspace.IsValidLayout(layout) {
		return createWorkspaceResult{}, fmt.Errorf("invalid layout %q (valid: %v)", opts.Layout, workspace.ValidLayouts())
	}

	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return createWorkspaceResult{}, fmt.Errorf("resolve path: %w", err)
	}

	store, err := newStore()
	if err != nil {
		return createWorkspaceResult{}, err
	}
	client := newTmuxClient()
	mgr := newManager(store, client)
	mode := workspace.CreateMode(opts.Mode)

	switch mode {
	case workspace.CreateModeNewWorktree:
		ws, err := mgr.CreateWithWorktree(ctx, workspace.ManagedWorkspaceRequest{
			Title:      opts.Title,
			AgentType:  agentType,
			RepoRoot:   absPath,
			Branch:     opts.Branch,
			BaseBranch: opts.BaseBranch,
			Layout:     layout,
		})
		if err != nil {
			return createWorkspaceResult{}, fmt.Errorf("create worktree workspace: %w", err)
		}
		return createWorkspaceResult{Workspace: ws, Workspaces: []workspace.Workspace{*ws}}, nil
	case workspace.CreateModeExperimentRun:
		result, err := mgr.CreateExperiment(ctx, workspace.ExperimentRequest{
			Title:          opts.Title,
			Prompt:         opts.Prompt,
			RepoRoot:       absPath,
			BaseBranch:     opts.BaseBranch,
			CandidateCount: opts.CandidateCount,
			AgentStrategy:  workspace.ExperimentAgentStrategy(opts.AgentStrategy),
			AgentType:      agentType,
			Layout:         layout,
		})
		if err != nil {
			return createWorkspaceResult{}, fmt.Errorf("create experiment: %w", err)
		}
		workspaces := make([]workspace.Workspace, 0, len(result.Workspaces))
		for _, ws := range result.Workspaces {
			if ws != nil {
				workspaces = append(workspaces, *ws)
			}
		}
		return createWorkspaceResult{Experiment: result.Experiment, Workspaces: workspaces, Broadcast: result.Broadcast}, nil
	case workspace.CreateModeExistingCheckout:
		ws, err := mgr.CreateStandalone(ctx, workspace.StandaloneWorkspaceRequest{
			Title:        opts.Title,
			AgentType:    agentType,
			CheckoutPath: absPath,
			Layout:       layout,
		})
		if err != nil {
			return createWorkspaceResult{}, fmt.Errorf("create workspace: %w", err)
		}
		return createWorkspaceResult{Workspace: ws, Workspaces: []workspace.Workspace{*ws}}, nil
	default:
		return createWorkspaceResult{}, fmt.Errorf("unsupported create mode %q", opts.Mode)
	}
}

func defaultCreateWorkspaceOptions(title string) createWorkspaceOptions {
	return createWorkspaceOptions{
		Title:          title,
		Path:           ".",
		Agent:          cfg.Defaults.Agent,
		Layout:         cfg.Defaults.Layout,
		Mode:           string(workspace.CreateModeExistingCheckout),
		CandidateCount: 2,
		AgentStrategy:  string(workspace.ExperimentAgentAllSupported),
	}
}
