package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/cliapi"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type paneListerReader interface {
	ListPanes(ctx context.Context, session string) ([]tmux.PaneInfo, error)
	CapturePane(ctx context.Context, target string, lines int) (string, error)
}

type paneController interface {
	paneListerReader
	SendKeys(ctx context.Context, target string, keys string, opts tmux.SendOptions) error
}

func newPaneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Scriptable tmux pane operations",
	}
	cmd.AddCommand(newPaneListCmd(), newPaneReadCmd(), newPaneSendCmd(), newPaneRunCmd())
	return cmd
}

func newPaneListCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "list <workspace>",
		Short: "List workspace panes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneList(cmd.Context(), cmd.OutOrStdout(), args[0], jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func newPaneReadCmd() *cobra.Command {
	var pane string
	var lines int
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "read <workspace>",
		Short: "Read recent pane content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneRead(cmd.Context(), cmd.OutOrStdout(), args[0], pane, lines, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "agent", "pane role or tmux pane target")
	cmd.Flags().IntVar(&lines, "lines", 80, "number of lines to capture")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return cmd
}

func runPaneList(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	workspaces, err := store.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}
	ws, err := resolveWorkspace(workspaces, workspaceTarget)
	if err != nil {
		return err
	}
	client := newTmuxClient()
	panes, err := workspacePanes(ctx, client, ws)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(out, cliapi.PaneListResponse{Workspace: cliapi.NewWorkspace(ws), Panes: panes})
	}
	for _, pane := range panes {
		label := pane.Target
		if pane.Role != "" {
			label = pane.Role + "\t" + pane.Target
		}
		if _, err := fmt.Fprintf(out, "%s\t%d\t%d\n", label, pane.Width, pane.Height); err != nil {
			return err
		}
	}
	return nil
}

func runPaneRead(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget, pane string, lines int, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	reader := newTmuxClient()
	return runPaneReadWithDeps(ctx, out, store, reader, workspaceTarget, pane, lines, jsonOutput)
}

func runPaneReadWithDeps(ctx context.Context, out interface{ Write([]byte) (int, error) }, store workspaceStore, reader paneListerReader, workspaceTarget, pane string, lines int, jsonOutput bool) error {
	if lines <= 0 {
		return fmt.Errorf("lines must be greater than 0")
	}
	workspaces, err := store.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}
	ws, err := resolveWorkspace(workspaces, workspaceTarget)
	if err != nil {
		return err
	}
	target, err := resolvePaneTarget(ws, pane)
	if err != nil {
		return err
	}
	content, err := reader.CapturePane(ctx, target, lines)
	if err != nil {
		return fmt.Errorf("read pane %q: %w", target, err)
	}
	if jsonOutput {
		return writeJSON(out, cliapi.PaneReadResponse{
			Workspace: cliapi.NewWorkspace(ws),
			Pane:      pane,
			Target:    target,
			Lines:     lines,
			Content:   content,
		})
	}
	_, err = fmt.Fprintln(out, content)
	return err
}

func workspacePanes(ctx context.Context, lister paneListerReader, ws workspace.Workspace) ([]cliapi.Pane, error) {
	live, err := lister.ListPanes(ctx, workspaceSessionName(ws))
	if err != nil {
		return nil, fmt.Errorf("list panes for workspace %q: %w", ws.Title, err)
	}
	byTarget := make(map[string]tmux.PaneInfo, len(live))
	for _, pane := range live {
		byTarget[pane.ID] = pane
	}

	roles := make([]string, 0, len(ws.PaneTargets))
	for role := range ws.PaneTargets {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	result := make([]cliapi.Pane, 0, len(live))
	seen := make(map[string]struct{}, len(live))
	for _, role := range roles {
		target := ws.PaneTargets[role]
		pane := cliapi.Pane{Role: role, Target: target}
		if info, ok := byTarget[target]; ok {
			pane.Width = info.Width
			pane.Height = info.Height
		}
		result = append(result, pane)
		seen[target] = struct{}{}
	}
	for _, info := range live {
		if _, ok := seen[info.ID]; ok {
			continue
		}
		result = append(result, cliapi.Pane{Target: info.ID, Width: info.Width, Height: info.Height})
	}
	return result, nil
}

func newPaneSendCmd() *cobra.Command {
	var pane string
	var text string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "send <workspace>",
		Short: "Send text to a workspace pane",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneSend(cmd.Context(), cmd.OutOrStdout(), args[0], pane, text, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "agent", "pane role or tmux pane target")
	cmd.Flags().StringVar(&text, "text", "", "text to send")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("text")
	return cmd
}

func newPaneRunCmd() *cobra.Command {
	var pane string
	var command string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "run <workspace>",
		Short: "Run a command in a workspace pane",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneRun(cmd.Context(), cmd.OutOrStdout(), args[0], pane, command, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&pane, "pane", "shell", "pane role or tmux pane target")
	cmd.Flags().StringVar(&command, "command", "", "command to run")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("command")
	return cmd
}

func runPaneSend(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget, pane, text string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	return runPaneSendWithDeps(ctx, out, store, client, workspaceTarget, pane, text, "send", jsonOutput)
}

func runPaneRun(ctx context.Context, out interface{ Write([]byte) (int, error) }, workspaceTarget, pane, command string, jsonOutput bool) error {
	store, err := newStore()
	if err != nil {
		return err
	}
	client := newTmuxClient()
	return runPaneSendWithDeps(ctx, out, store, client, workspaceTarget, pane, command, "run", jsonOutput)
}

func runPaneSendWithDeps(ctx context.Context, out interface{ Write([]byte) (int, error) }, store workspaceStore, controller paneController, workspaceTarget, pane, text, action string, jsonOutput bool) error {
	if strings.TrimSpace(text) == "" {
		if action == "run" {
			return fmt.Errorf("run command cannot be empty")
		}
		return fmt.Errorf("send text cannot be empty")
	}
	workspaces, err := store.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}
	ws, err := resolveWorkspace(workspaces, workspaceTarget)
	if err != nil {
		return err
	}
	target, err := resolvePaneTarget(ws, pane)
	if err != nil {
		return err
	}
	opts := tmux.SendOptions{}
	if pane == "agent" || target == ws.PaneTargets["agent"] {
		opts = workspace.AgentSendOptions(ws, text)
	}
	if err := controller.SendKeys(ctx, target, text, opts); err != nil {
		return fmt.Errorf("%s pane %q: %w", action, target, err)
	}
	if jsonOutput {
		return writeJSON(out, cliapi.PaneActionResponse{
			Workspace: cliapi.NewWorkspace(ws),
			Pane:      pane,
			Target:    target,
			Action:    action,
		})
	}
	_, err = fmt.Fprintf(out, "%s sent to %s\n", action, target)
	return err
}
