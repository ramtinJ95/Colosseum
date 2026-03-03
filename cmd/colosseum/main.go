package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/tui"
	"github.com/ramtinj/colosseum/internal/workspace"
)

const sessionPrefix = "colo-"

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "colosseum",
	Short: "AI agent workspace manager",
	Long:  "A TUI for managing parallel AI coding agents across git worktrees, built on tmux.",
	RunE:  runDashboard,
}

var newCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	RunE:  runList,
}

var attachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to a workspace's tmux session",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

var (
	flagPath   string
	flagAgent  string
	flagBranch string
	flagLayout string
)

func init() {
	newCmd.Flags().StringVarP(&flagPath, "path", "p", ".", "project directory path")
	newCmd.Flags().StringVarP(&flagAgent, "agent", "a", "claude", "agent type (claude, codex, gemini, opencode, aider)")
	newCmd.Flags().StringVarP(&flagBranch, "branch", "b", "", "git branch name")
	newCmd.Flags().StringVarP(&flagLayout, "layout", "l", "agent-shell", "pane layout (agent, agent-shell, agent-shell-logs)")

	rootCmd.AddCommand(newCmd, listCmd, attachCmd, deleteCmd)
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
	return tmux.NewClient(tmux.NewExecCommander())
}

func runDashboard(_ *cobra.Command, _ []string) error {
	store := newStore()
	client := newTmuxClient()

	detector := status.NewDetector(client, 50)
	poller := status.NewPoller(detector, store, 1500*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go poller.Run(ctx)

	app := tui.NewApp(store, poller, detector)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := p.Run()
	return err
}

func runNew(_ *cobra.Command, args []string) error {
	name := args[0]
	agentType := agent.AgentType(flagAgent)

	if _, ok := agent.Get(agentType); !ok {
		return fmt.Errorf("unknown agent type: %s (available: %v)", flagAgent, agent.Available())
	}

	absPath, err := filepath.Abs(flagPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	store := newStore()
	client := newTmuxClient()
	mgr := workspace.NewManager(store, client, sessionPrefix)

	layout := workspace.LayoutType(flagLayout)
	ws, err := mgr.Create(context.Background(), name, agentType, absPath, flagBranch, layout)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	fmt.Printf("Created workspace %q (session: %s)\n", ws.Title, ws.SessionName)
	return nil
}

func runList(_ *cobra.Command, _ []string) error {
	store := newStore()
	workspaces, err := store.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces. Create one with: colosseum new <name>")
		return nil
	}

	for _, ws := range workspaces {
		branch := ""
		if ws.Branch != "" {
			branch = fmt.Sprintf(" [%s]", ws.Branch)
		}
		fmt.Printf("  %s %s%s (%s · %s)\n",
			statusIcon(ws.Status), ws.Title, branch,
			ws.AgentType, ws.Status)
	}
	return nil
}

func runAttach(_ *cobra.Command, args []string) error {
	name := args[0]
	store := newStore()
	client := newTmuxClient()
	mgr := workspace.NewManager(store, client, sessionPrefix)

	workspaces, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	for _, ws := range workspaces {
		if ws.Title == name {
			return mgr.SwitchTo(context.Background(), ws.ID)
		}
	}

	return fmt.Errorf("workspace %q not found", name)
}

func runDelete(_ *cobra.Command, args []string) error {
	name := args[0]
	store := newStore()
	client := newTmuxClient()
	mgr := workspace.NewManager(store, client, sessionPrefix)

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

func statusIcon(s agent.Status) string {
	switch s {
	case agent.StatusWorking:
		return "●"
	case agent.StatusWaiting:
		return "◉"
	case agent.StatusIdle:
		return "○"
	case agent.StatusStopped:
		return "■"
	case agent.StatusError:
		return "✗"
	default:
		return "?"
	}
}
