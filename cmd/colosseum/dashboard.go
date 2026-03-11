package main

import (
	"context"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tui"
)

func runDashboard(_ *cobra.Command, _ []string) error {
	bootstrap, err := newDashboardBootstrap(newTmuxClient())
	if err != nil {
		return err
	}
	handled, err := bootstrap.Bootstrap(context.Background())
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	return runDashboardProgram()
}

func runDashboardProgram() (runErr error) {
	store := newStore()
	client := newTmuxClient()
	mgr := newManager(store, client)

	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	poller := status.NewPoller(detector, store, time.Duration(cfg.Status.PollIntervalMS)*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer func() {
		if err := restoreDashboardSession(context.Background(), client, os.Getenv); err != nil && runErr == nil {
			runErr = err
		}
	}()

	go poller.Run(ctx)

	app := tui.NewApp(store, mgr, poller, detector, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, runErr = p.Run()
	return runErr
}
