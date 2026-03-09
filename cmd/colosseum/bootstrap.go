package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/workspace"
	"github.com/ramtinj/colosseum/internal/worktrunk"
)

const (
	dashboardInternalEnv  = "COLOSSEUM_DASHBOARD_INTERNAL"
	dashboardSessionTitle = "dashboard"
)

type dashboardSessionController interface {
	SessionExists(ctx context.Context, name string) bool
	CurrentSession(ctx context.Context) (string, error)
	CreateDetachedSessionWithCommand(ctx context.Context, name string, startDir string, command []string) error
	SwitchSession(ctx context.Context, name string) error
	AttachSession(ctx context.Context, name string) error
}

type dashboardBootstrap struct {
	client         dashboardSessionController
	sessionName    string
	currentDir     string
	executablePath string
	getenv         func(string) string
}

func stateDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "colosseum")
}

func newStore() *workspace.Store {
	dir := stateDir()
	os.MkdirAll(dir, 0o755)
	return workspace.NewStore(filepath.Join(dir, "workspaces.json"))
}

func newTmuxClient() *tmux.Client {
	c := tmux.NewClient(tmux.NewExecCommander())
	c.SessionPrefix = cfg.Tmux.SessionPrefix
	c.ReturnKey = cfg.Tmux.ReturnKey
	return c
}

func newManager(store *workspace.Store, client *tmux.Client) *workspace.Manager {
	return workspace.NewManager(store, client, worktrunk.NewClient(), cfg.Tmux.SessionPrefix)
}

func newDashboardBootstrap(client *tmux.Client) (dashboardBootstrap, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return dashboardBootstrap{}, fmt.Errorf("resolve working directory: %w", err)
	}
	executablePath, err := os.Executable()
	if err != nil {
		return dashboardBootstrap{}, fmt.Errorf("resolve colosseum executable: %w", err)
	}
	return dashboardBootstrap{
		client:         client,
		sessionName:    dashboardSessionName(),
		currentDir:     currentDir,
		executablePath: executablePath,
		getenv:         os.Getenv,
	}, nil
}

func dashboardSessionName() string {
	return cfg.Tmux.SessionPrefix + dashboardSessionTitle
}

func workspaceSessionName(ws workspace.Workspace) string {
	if ws.SessionName != "" {
		return ws.SessionName
	}
	return cfg.Tmux.SessionPrefix + ws.Title
}

func (b dashboardBootstrap) Bootstrap(ctx context.Context) (bool, error) {
	if strings.TrimSpace(b.getenv(dashboardInternalEnv)) == "1" {
		return false, nil
	}

	insideTmux := strings.TrimSpace(b.getenv("TMUX")) != ""
	if insideTmux {
		currentSession, err := b.client.CurrentSession(ctx)
		if err != nil {
			return false, fmt.Errorf("detect current tmux session: %w", err)
		}
		if currentSession == b.sessionName {
			return false, nil
		}
	}

	if !b.client.SessionExists(ctx, b.sessionName) {
		if err := b.client.CreateDetachedSessionWithCommand(ctx, b.sessionName, b.currentDir, b.command()); err != nil {
			return false, fmt.Errorf("create dashboard session %q: %w", b.sessionName, err)
		}
	}

	if insideTmux {
		if err := b.client.SwitchSession(ctx, b.sessionName); err != nil {
			return true, fmt.Errorf("switch to dashboard session %q: %w", b.sessionName, err)
		}
		return true, nil
	}

	if err := b.client.AttachSession(ctx, b.sessionName); err != nil {
		return true, fmt.Errorf("attach to dashboard session %q: %w", b.sessionName, err)
	}
	return true, nil
}

func (b dashboardBootstrap) command() []string {
	return []string{"env", dashboardInternalEnv + "=1", b.executablePath}
}
