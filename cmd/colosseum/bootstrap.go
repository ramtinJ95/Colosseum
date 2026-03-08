package main

import (
	"os"
	"path/filepath"

	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/workspace"
	"github.com/ramtinj/colosseum/internal/worktrunk"
)

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
