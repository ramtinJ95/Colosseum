package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <name>",
		Short: "Attach to a workspace's tmux session",
		Args:  cobra.ExactArgs(1),
		RunE:  runAttach,
	}
}

func runAttach(_ *cobra.Command, args []string) error {
	name := args[0]
	store := newStore()
	client := newTmuxClient()
	mgr := newManager(store, client)

	workspaces, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	for _, ws := range workspaces {
		if ws.Title == name {
			if strings.TrimSpace(os.Getenv("TMUX")) == "" {
				return client.AttachSession(context.Background(), workspaceSessionName(ws))
			}
			return mgr.SwitchTo(context.Background(), ws.ID)
		}
	}

	return fmt.Errorf("workspace %q not found", name)
}
