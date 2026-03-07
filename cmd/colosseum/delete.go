package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workspace",
		Args:  cobra.ExactArgs(1),
		RunE:  runDelete,
	}
}

func runDelete(_ *cobra.Command, args []string) error {
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
			if err := mgr.Delete(context.Background(), ws.ID); err != nil {
				return fmt.Errorf("delete workspace: %w", err)
			}
			fmt.Printf("Deleted workspace %q\n", name)
			return nil
		}
	}

	return fmt.Errorf("workspace %q not found", name)
}
