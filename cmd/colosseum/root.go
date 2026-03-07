package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "colosseum",
	Short: "AI agent workspace manager",
	Long:  "A tmux-native TUI for managing parallel AI coding agent workspaces.",
	RunE:  runDashboard,
}

var (
	flagPath   string
	flagAgent  string
	flagBranch string
	flagLayout string
	cfg        config.Config
)

func init() {
	var err error
	cfg, err = config.Load(config.DefaultPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(newNewCmd(), newListCmd(), newAttachCmd(), newDeleteCmd())
}
