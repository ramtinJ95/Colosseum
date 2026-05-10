package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/cliapi"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func newWaitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for workspace state or output",
	}
	cmd.AddCommand(newWaitStatusCmd(), newWaitOutputCmd())
	return cmd
}

func newWaitStatusCmd() *cobra.Command {
	var desired string
	var timeout time.Duration
	var interval time.Duration
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "status <workspace>",
		Short: "Wait for a workspace to reach a status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWaitStatus(cmd.Context(), cmd.OutOrStdout(), args[0], desired, timeout, interval, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&desired, "status", agent.StatusIdle.String(), "desired status")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "maximum time to wait")
	cmd.Flags().DurationVar(&interval, "interval", 1500*time.Millisecond, "poll interval")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func newWaitOutputCmd() *cobra.Command {
	var pane string
	var match string
	var lines int
	var timeout time.Duration
	var interval time.Duration
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "output <workspace>",
		Short: "Wait for pane output to match a regular expression",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWaitOutput(cmd.Context(), cmd.OutOrStdout(), args[0], pane, match, lines, timeout, interval, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "agent", "pane role or tmux pane target")
	cmd.Flags().StringVar(&match, "match", "", "regular expression to wait for")
	cmd.Flags().IntVar(&lines, "lines", 80, "number of lines to capture each poll")
	cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "maximum time to wait")
	cmd.Flags().DurationVar(&interval, "interval", 1*time.Second, "poll interval")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("match")
	return cmd
}

func runWaitStatus(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget, desiredValue string, timeout, interval time.Duration, jsonOutput bool) error {
	desired, err := parseAgentStatus(desiredValue)
	if err != nil {
		return err
	}
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	ws, got, elapsed, err := waitForWorkspaceStatus(ctx, store, detector, workspaceTarget, desired, timeout, interval)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(out, cliapi.WaitStatusResponse{
			Workspace: cliapi.NewWorkspace(ws),
			Status:    got.String(),
			Desired:   desired.String(),
			ElapsedMS: elapsed.Milliseconds(),
		})
	}
	_, err = fmt.Fprintf(out, "%s reached %s\n", ws.Title, got)
	return err
}

func waitForWorkspaceStatus(ctx context.Context, store workspaceStore, detector workspaceStatusDetector, workspaceTarget string, desired agent.Status, timeout, interval time.Duration) (workspace.Workspace, agent.Status, time.Duration, error) {
	if timeout <= 0 {
		return workspace.Workspace{}, agent.StatusUnknown, 0, fmt.Errorf("timeout must be greater than 0")
	}
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()

	check := func() (workspace.Workspace, agent.Status, error) {
		workspaces, reports, err := loadWorkspacesAndReports(store)
		if err != nil {
			return workspace.Workspace{}, agent.StatusUnknown, fmt.Errorf("list workspaces: %w", err)
		}
		ws, err := resolveWorkspace(workspaces, workspaceTarget)
		if err != nil {
			return workspace.Workspace{}, agent.StatusUnknown, err
		}
		current, err := detectWorkspaceStatusWithReports(ctx, ws, detector, reports)
		if err != nil {
			return workspace.Workspace{}, agent.StatusUnknown, err
		}
		ws.Status = current
		return ws, current, nil
	}

	var last workspace.Workspace
	var lastStatus agent.Status
	for {
		ws, current, err := check()
		if err != nil {
			return workspace.Workspace{}, agent.StatusUnknown, time.Since(started), err
		}
		last = ws
		lastStatus = current
		if current == desired {
			return ws, current, time.Since(started), nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return last, lastStatus, time.Since(started), fmt.Errorf("timed out waiting for workspace %q to reach %s; last status was %s", workspaceTarget, desired, lastStatus)
		case <-timer.C:
		}
	}
}

func runWaitOutput(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget, pane, match string, lines int, timeout, interval time.Duration, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	reader := newTmuxClient()
	ws, target, elapsed, err := waitForPaneOutput(ctx, store, reader, workspaceTarget, pane, match, lines, timeout, interval)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(out, cliapi.WaitOutputResponse{
			Workspace: cliapi.NewWorkspace(ws),
			Pane:      pane,
			Target:    target,
			Match:     match,
			ElapsedMS: elapsed.Milliseconds(),
		})
	}
	_, err = fmt.Fprintf(out, "%s matched %q in %s\n", target, match, elapsed.Round(time.Millisecond))
	return err
}

func waitForPaneOutput(ctx context.Context, store workspaceStore, reader paneListerReader, workspaceTarget, pane, match string, lines int, timeout, interval time.Duration) (workspace.Workspace, string, time.Duration, error) {
	if lines <= 0 {
		return workspace.Workspace{}, "", 0, fmt.Errorf("lines must be greater than 0")
	}
	if timeout <= 0 {
		return workspace.Workspace{}, "", 0, fmt.Errorf("timeout must be greater than 0")
	}
	if interval <= 0 {
		interval = 1 * time.Second
	}
	pattern, err := regexp.Compile(match)
	if err != nil {
		return workspace.Workspace{}, "", 0, fmt.Errorf("compile match regex: %w", err)
	}
	workspaces, err := store.List()
	if err != nil {
		return workspace.Workspace{}, "", 0, fmt.Errorf("list workspaces: %w", err)
	}
	ws, err := resolveWorkspace(workspaces, workspaceTarget)
	if err != nil {
		return workspace.Workspace{}, "", 0, err
	}
	target, err := resolvePaneTarget(ws, pane)
	if err != nil {
		return workspace.Workspace{}, "", 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	for {
		content, err := reader.CapturePane(ctx, target, lines)
		if err != nil {
			return workspace.Workspace{}, "", time.Since(started), fmt.Errorf("read pane %q: %w", target, err)
		}
		if pattern.MatchString(content) {
			return ws, target, time.Since(started), nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ws, target, time.Since(started), fmt.Errorf("timed out waiting for pane %q output to match %q", target, match)
		case <-timer.C:
		}
	}
}
