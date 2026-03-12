# Colosseum — Complete Code Walkthrough

*2026-03-09T21:21:19Z by Showboat 0.6.1*
<!-- showboat-id: 9b84bdc9-4506-4ef7-aa10-a46f846e6c40 -->

Colosseum is a Go TUI for managing parallel AI coding agents across git worktrees, built on tmux and Bubble Tea. It lets you spin up multiple isolated workspaces — each with its own tmux session, git worktree, and AI agent (Claude, Codex, OpenCode, etc.) — monitor their status in real-time from a central dashboard, and broadcast prompts to many agents at once. This walkthrough traces the code from the entry point down through every layer.

## 1. Project Structure

The codebase is organized into a thin CLI entry point and six internal packages:

```bash
find . -type f -name '*.go' \! -path './.claude/*' \! -path './vendor/*' | sort | sed 's|^\./||'
```

```output
cmd/colosseum/attach.go
cmd/colosseum/bootstrap.go
cmd/colosseum/broadcast.go
cmd/colosseum/broadcast_test.go
cmd/colosseum/dashboard.go
cmd/colosseum/delete.go
cmd/colosseum/list.go
cmd/colosseum/main.go
cmd/colosseum/new.go
cmd/colosseum/root.go
internal/agent/agent.go
internal/agent/aider.go
internal/agent/claude.go
internal/agent/codex.go
internal/agent/gemini.go
internal/agent/opencode.go
internal/agent/patterns.go
internal/agent/registry.go
internal/agent/registry_test.go
internal/config/config.go
internal/config/config_test.go
internal/status/detector.go
internal/status/detector_test.go
internal/status/normalize.go
internal/status/normalize_test.go
internal/status/poller.go
internal/status/poller_test.go
internal/status/refresh.go
internal/status/types.go
internal/tmux/commander.go
internal/tmux/commander_mock.go
internal/tmux/format.go
internal/tmux/format_test.go
internal/tmux/pane.go
internal/tmux/pane_test.go
internal/tmux/session.go
internal/tmux/session_test.go
internal/tui/app.go
internal/tui/app_test.go
internal/tui/dialog/broadcast.go
internal/tui/dialog/broadcast_test.go
internal/tui/dialog/delete.go
internal/tui/dialog/delete_test.go
internal/tui/dialog/help.go
internal/tui/dialog/keymap.go
internal/tui/dialog/new_workspace.go
internal/tui/dialog/new_workspace_test.go
internal/tui/dialog/theme_test.go
internal/tui/help.go
internal/tui/keys.go
internal/tui/preview/model.go
internal/tui/preview/model_test.go
internal/tui/preview/view.go
internal/tui/sidebar/model.go
internal/tui/sidebar/model_test.go
internal/tui/sidebar/view.go
internal/tui/theme/theme.go
internal/workspace/git.go
internal/workspace/layout.go
internal/workspace/layout_test.go
internal/workspace/manager.go
internal/workspace/manager_test.go
internal/workspace/storage.go
internal/workspace/storage_test.go
internal/workspace/workspace.go
internal/worktrunk/client.go
internal/worktrunk/client_test.go
```

- **cmd/colosseum/** — CLI entry point (Cobra commands)
- **internal/config/** — TOML configuration with defaults
- **internal/agent/** — Agent type definitions and regex-based detection patterns
- **internal/tmux/** — Low-level tmux command abstraction (os/exec)
- **internal/workspace/** — Workspace model, JSON persistence, Manager lifecycle
- **internal/worktrunk/** — Git worktree management via the external `wt` tool
- **internal/status/** — Background status detection engine (ANSI stripping, pattern matching, polling)
- **internal/tui/** — Bubble Tea TUI (app, sidebar, preview, dialogs, theme, keybindings)

## 2. Entry Point — main.go and root.go

The program starts in `cmd/colosseum/main.go`, which simply calls `rootCmd.Execute()`. The root command is defined in `root.go` using Cobra:

```bash
sed -n '1,37p' cmd/colosseum/root.go
```

```output
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

	rootCmd.AddCommand(newNewCmd(), newListCmd(), newAttachCmd(), newBroadcastCmd(), newDeleteCmd())
}
```

The `init()` function loads the TOML configuration (falling back to defaults if the file is absent) and registers five subcommands: `new`, `list`, `attach`, `broadcast`, and `delete`. Running `colosseum` with no subcommand invokes `runDashboard`, which launches the full TUI.

The `bootstrap.go` file wires up shared infrastructure — the JSON store, the tmux client, and the workspace manager — that both the TUI and CLI subcommands use:

```bash
cat cmd/colosseum/bootstrap.go
```

```output
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
```

Key points: state lives in `~/.config/colosseum/workspaces.json`, the tmux client wraps `os/exec` with a configurable session prefix (default `colo-`), and the workspace manager receives both the store and tmux client as dependencies.

## 3. Configuration — internal/config

Configuration is loaded from `~/.config/colosseum/config.toml`. If the file does not exist, sensible defaults are used. The config covers:

- **Defaults**: default agent (`claude`) and layout (`agent-shell`)
- **Status**: poll interval (1500ms) and capture lines (50)
- **UI**: preview refresh (750ms), sidebar width bounds (30–50)
- **Tmux**: session prefix (`colo-`) and return key (`e`)
- **Keys**: vim-style keybindings (j/k/h/l, etc.)
- **Theme**: 256-color values for every UI element

```bash
sed -n '82,139p' internal/config/config.go
```

```output
func Default() Config {
	return Config{
		Defaults: DefaultsConfig{
			Agent:  "claude",
			Layout: "agent-shell",
		},
		Status: StatusConfig{
			PollIntervalMS: 1500,
			CaptureLines:   50,
		},
		UI: UIConfig{
			PreviewRefreshMS: 750,
			SidebarMinWidth:  30,
			SidebarMaxWidth:  50,
		},
		Tmux: TmuxConfig{
			SessionPrefix: "colo-",
			ReturnKey:     "e",
		},
		Keys: KeysConfig{
			Up:        "k",
			Down:      "j",
			Enter:     "enter",
			New:       "n",
			Delete:    "d",
			PaneLeft:  "h",
			PaneRight: "l",
			Broadcast: "b",
			Diff:      "D",
			Rename:    "r",
			Filter:    "/",
			Tab:       "tab",
			MarkRead:  "m",
			JumpNext:  "J",
			Restart:   "R",
			Stop:      "s",
			Help:      "?",
			Quit:      "q",
		},
		Theme: ThemeConfig{
			Border:     "62",
			AppTitle:   "99",
			SelectedFG: "99",
			SelectedBG: "236",
			Normal:     "252",
			Working:    "82",
			Waiting:    "220",
			Idle:       "245",
			Stopped:    "240",
			Error:      "196",
			Branch:     "109",
			AgentName:  "140",
			HelpKey:    "99",
			HelpDesc:   "245",
			Dim:        "240",
		},
	}
}
```

The `Load` function reads TOML into a pre-filled default struct, then validates key bindings for duplicates. This means users only need to override the specific values they want to change.

## 4. Agent Definitions — internal/agent

The agent package defines what AI coding agents Colosseum knows about. Each agent is described by an `AgentDef` struct:

```bash
cat internal/agent/agent.go
```

```output
package agent

import (
	"regexp"
	"time"
)

type AgentType string

const (
	Claude   AgentType = "claude"
	Codex    AgentType = "codex"
	Gemini   AgentType = "gemini"
	OpenCode AgentType = "opencode"
	Aider    AgentType = "aider"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusWorking
	StatusWaiting
	StatusIdle
	StatusStopped
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusWorking:
		return "Working"
	case StatusWaiting:
		return "Waiting"
	case StatusIdle:
		return "Idle"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

type AgentDef struct {
	Name                              AgentType
	Binary                            string
	LaunchFlags                       []string
	YoloFlags                         []string
	InputDelay                        time.Duration
	PasteSingleLine                   bool
	DisableBracketedPasteForMultiline bool
	IgnorePatterns                    []*regexp.Regexp
	WorkingPatterns                   []*regexp.Regexp
	WaitingPatterns                   []*regexp.Regexp
	IdlePatterns                      []*regexp.Regexp
	ErrorPatterns                     []*regexp.Regexp
}
```

The five status levels — Working, Waiting, Idle, Stopped, Error — drive the entire status detection system. Each `AgentDef` carries four sets of regex patterns that determine how terminal output maps to these statuses.

### Agent Registry

Agents self-register via `init()` functions. Here is the registry and then one example agent definition — Claude:

```bash
cat internal/agent/registry.go
```

```output
package agent

import "sort"

var registry = make(map[AgentType]*AgentDef)

var supportedAgents = []AgentType{Claude, Codex, OpenCode}

func Register(def *AgentDef) {
	registry[def.Name] = def
}

func Get(name AgentType) (*AgentDef, bool) {
	def, ok := registry[name]
	return def, ok
}

func Available() []AgentType {
	types := make([]AgentType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func Supported() []AgentType {
	types := make([]AgentType, len(supportedAgents))
	copy(types, supportedAgents)
	return types
}

func IsSupported(name AgentType) bool {
	for _, supported := range supportedAgents {
		if name == supported {
			return true
		}
	}
	return false
}
```

Note the distinction between `Available()` (everything registered, including experimental agents like Aider and Gemini) and `Supported()` (the vetted subset: Claude, Codex, OpenCode). The CLI validation uses `IsSupported()` to gate workspace creation.

### Claude Agent Definition

```bash
cat internal/agent/claude.go
```

```output
package agent

import (
	"regexp"
	"time"
)

func init() {
	Register(&AgentDef{
		Name:                              Claude,
		Binary:                            "claude",
		LaunchFlags:                       []string{},
		YoloFlags:                         []string{"--dangerously-skip-permissions"},
		InputDelay:                        100 * time.Millisecond,
		DisableBracketedPasteForMultiline: true,
		IgnorePatterns: []*regexp.Regexp{
			regexp.MustCompile(`Tokens:.*Remaining:`),
			regexp.MustCompile(`^\s*Opus .* \| .*`),
			regexp.MustCompile(`^[\s─▪]+$`),
			regexp.MustCompile(`^\s*--\s+(INSERT|NORMAL)\s+--`),
		},
		WorkingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\(esc to interrupt\)`),
			BrailleSpinner,
			regexp.MustCompile(`\(th(?:inking|ought)\b`),
			regexp.MustCompile(`\(ctrl\+o to expand\)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(yes,?\s*allow|allow\s*(once|always))`),
			regexp.MustCompile(`(?i)permission`),
			ChoiceMenuPattern,
			ChoicePromptPattern,
		},
		IdlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^>\s*$`),
			regexp.MustCompile(`^\$\s*$`),
			regexp.MustCompile(`^❯`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
```

Each agent definition captures the full personality of how that agent behaves in a terminal:

- **Binary / LaunchFlags**: how to start the agent (e.g. `claude`)
- **YoloFlags**: auto-approve mode flags (e.g. `--dangerously-skip-permissions`)
- **InputDelay**: delay after sending keys, preventing race conditions
- **IgnorePatterns**: lines to strip before detection (status bars, decorative lines)
- **WorkingPatterns**: regex that means "agent is actively processing" (spinners, "esc to interrupt")
- **WaitingPatterns**: regex that means "agent wants human input" (permission prompts, choice menus)
- **IdlePatterns**: regex that means "agent is at a fresh prompt" (`>`, `$`, `❯`)
- **ErrorPatterns**: regex that means something is wrong (rate limits, panics, auth failures)

### Shared Patterns

```bash
cat internal/agent/patterns.go
```

```output
package agent

import "regexp"

var (
	BrailleSpinner      = regexp.MustCompile(`[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	CommonPromptChars   = regexp.MustCompile(`^\s*[>$❯]\s*$`)
	ChoiceMenuPattern   = regexp.MustCompile(`(?m)^\s*❯\s*\d+\.`)
	ChoicePromptPattern = regexp.MustCompile(`(?i)(which .* (prefer|use)|which approach|which testing framework|what .* want to tackle|how would you like|select (one|an option)|choose (one|an option))`)
	RateLimitPattern    = regexp.MustCompile(`(?i)(rate.?limit|429|too many requests)`)
	PanicPattern        = regexp.MustCompile(`(?i)(panic:|fatal error:|segmentation fault)`)
	AuthErrorPattern    = regexp.MustCompile(`(?i)(unauthorized|authentication failed|invalid.*api.*key|EAUTH)`)
)
```

These shared patterns are reused across multiple agent definitions. `BrailleSpinner` detects the Unicode braille characters (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏) that CLI tools use for animated spinners — a reliable working indicator.

## 5. Tmux Abstraction — internal/tmux

The tmux package wraps all tmux interaction behind a `Commander` interface, making it testable:

### Commander Interface and os/exec Implementation

```bash
sed -n '15,17p' internal/tmux/commander.go
```

```output
type Commander interface {
	Run(ctx context.Context, args ...string) (string, error)
}
```

The `Commander` interface has a single method — `Run` — that takes tmux arguments and returns stdout. `ExecCommander` implements this with `os/exec`, adding a timeout (default 5s) and structured error reporting via `TmuxError`. `MockCommander` provides a test double with pre-programmed responses.

### Session Management (session.go)

The `Client` struct holds a `Commander`, a session prefix, and a return key. It provides high-level tmux operations:

```bash
grep -n 'func (c \*Client)' internal/tmux/session.go internal/tmux/pane.go
```

```output
internal/tmux/session.go:26:func (c *Client) CreateSession(ctx context.Context, name string, startDir string) (string, error) {
internal/tmux/session.go:35:func (c *Client) KillSession(ctx context.Context, name string) error {
internal/tmux/session.go:43:func (c *Client) SessionExists(ctx context.Context, name string) bool {
internal/tmux/session.go:48:func (c *Client) ListSessions(ctx context.Context) ([]string, error) {
internal/tmux/session.go:68:func (c *Client) SwitchClient(ctx context.Context, name string) error {
internal/tmux/session.go:93:func (c *Client) currentSession(ctx context.Context) (string, error) {
internal/tmux/pane.go:22:func (c *Client) SplitWindow(ctx context.Context, session string, horizontal bool, startDir string) (string, error) {
internal/tmux/pane.go:41:func (c *Client) CapturePane(ctx context.Context, target string, lines int) (string, error) {
internal/tmux/pane.go:53:func (c *Client) CapturePaneTitle(ctx context.Context, target string) (string, error) {
internal/tmux/pane.go:63:func (c *Client) SendKeys(ctx context.Context, target string, keys string, opts SendOptions) error {
internal/tmux/pane.go:79:func (c *Client) SendLiteralKeys(ctx context.Context, target string, text string) error {
internal/tmux/pane.go:87:func (c *Client) pasteBuffer(ctx context.Context, target string, text string, opts SendOptions) error {
internal/tmux/pane.go:109:func (c *Client) ResizePane(ctx context.Context, target string, width, height int) error {
internal/tmux/pane.go:121:func (c *Client) ListPanes(ctx context.Context, session string) ([]PaneInfo, error) {
```

The critical flow for the TUI: when you press Enter on a workspace, `SwitchClient` first binds a tmux key (configurable, default `prefix+e`) that takes you back to the dashboard session, then switches to the workspace session. This is how the "return to dashboard" feature works — it dynamically rebinds a tmux prefix key.

### SendKeys and Paste Buffer (pane.go)

The `SendKeys` method has sophisticated input handling for different agents:

```bash
sed -n '63,107p' internal/tmux/pane.go
```

```go
func (c *Client) SendKeys(ctx context.Context, target string, keys string, opts SendOptions) error {
	if opts.ForcePaste || strings.Contains(keys, "\n") {
		return c.pasteBuffer(ctx, target, keys, opts)
	}
	if _, err := c.Commander.Run(ctx, "send-keys", "-t", target, "-l", keys); err != nil {
		return fmt.Errorf("send keys to %q: %w", target, err)
	}
	if opts.InputDelay > 0 {
		time.Sleep(opts.InputDelay)
	}
	if _, err := c.Commander.Run(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return fmt.Errorf("send enter to %q: %w", target, err)
	}
	return nil
}

func (c *Client) SendLiteralKeys(ctx context.Context, target string, text string) error {
	_, err := c.Commander.Run(ctx, "send-keys", "-t", target, "-l", text)
	if err != nil {
		return fmt.Errorf("send literal keys to %q: %w", target, err)
	}
	return nil
}

func (c *Client) pasteBuffer(ctx context.Context, target string, text string, opts SendOptions) error {
	bufferName := fmt.Sprintf("colosseum-%d", time.Now().UnixNano())
	if _, err := c.Commander.Run(ctx, "set-buffer", "-b", bufferName, "--", text); err != nil {
		return fmt.Errorf("set paste buffer for %q: %w", target, err)
	}
	args := []string{"paste-buffer", "-d"}
	if !opts.DisableBracketedPaste {
		args = append(args, "-p")
	}
	args = append(args, "-r", "-b", bufferName, "-t", target)
	if _, err := c.Commander.Run(ctx, args...); err != nil {
		return fmt.Errorf("paste buffer into %q: %w", target, err)
	}
	if opts.InputDelay > 0 {
		time.Sleep(opts.InputDelay)
	}
	if _, err := c.Commander.Run(ctx, "send-keys", "-t", target, "Enter"); err != nil {
		return fmt.Errorf("send enter to %q: %w", target, err)
	}
	return nil
}
```

Two paths for input: single-line uses `send-keys -l` (literal), multi-line or `ForcePaste` uses tmux paste buffers with a unique nanoscond-based name (`colosseum-<timestamp>`). The `-p` flag enables bracketed paste unless the agent definition says otherwise (Claude needs it disabled for multiline). The `-d` flag auto-deletes the buffer after pasting. An optional `InputDelay` between the text and the Enter key prevents race conditions.

## 6. Workspace Model — internal/workspace

### Data Model (workspace.go)

The workspace package defines the central domain types. The `State` struct is what gets persisted to JSON:

```bash
sed -n '133,139p' internal/workspace/workspace.go
```

```output
type State struct {
	Workspaces   []Workspace  `json:"workspaces,omitempty"`
	Repositories []Repository `json:"repositories,omitempty"`
	Checkouts    []Checkout   `json:"checkouts,omitempty"`
	Experiments  []Experiment `json:"experiments,omitempty"`
	Evaluations  []Evaluation `json:"evaluations,omitempty"`
}
```

The state file has a rich relational model:

- **Repository**: a git repo by its root path
- **Checkout**: a specific branch/worktree within a repo (can be user-owned or Colosseum-managed)
- **Workspace**: a running tmux session with an agent, linked to a checkout
- **Experiment**: a group of workspaces with the same prompt but different agents or branches
- **Evaluation**: a judgment record (manual, vote, or agent-assisted)

The `Workspace` struct itself is the most important:

```bash
sed -n '89,106p' internal/workspace/workspace.go
```

```output
type Workspace struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	AgentType         agent.AgentType   `json:"agent_type"`
	ProjectPath       string            `json:"project_path"`
	Branch            string            `json:"branch"`
	BaseBranch        string            `json:"base_branch,omitempty"`
	RepositoryID      string            `json:"repository_id,omitempty"`
	CheckoutID        string            `json:"checkout_id,omitempty"`
	ExperimentID      string            `json:"experiment_id,omitempty"`
	CheckoutBackend   Backend           `json:"checkout_backend,omitempty"`
	CheckoutOwnership CheckoutOwnership `json:"checkout_ownership,omitempty"`
	Layout            LayoutType        `json:"layout"`
	Status            agent.Status      `json:"status"`
	SessionName       string            `json:"session_name"`
	PaneTargets       map[string]string `json:"pane_targets"`
	CreatedAt         time.Time         `json:"created_at"`
}
```

The `PaneTargets` map is crucial — it maps logical pane names (`agent`, `shell`, `logs`) to tmux pane IDs (like `%42`). This is how the status detector knows which pane to capture, and how broadcast sends prompts to the right pane.

### Layouts (layout.go)

Three tmux pane layouts control how many panes each workspace gets:

```bash
cat internal/workspace/layout.go
```

```output
package workspace

type LayoutType string

const (
	LayoutAgent          LayoutType = "agent"
	LayoutAgentShell     LayoutType = "agent-shell"
	LayoutAgentShellLogs LayoutType = "agent-shell-logs"
)

var validLayouts = []LayoutType{LayoutAgent, LayoutAgentShell, LayoutAgentShellLogs}

func ValidLayouts() []LayoutType {
	layouts := make([]LayoutType, len(validLayouts))
	copy(layouts, validLayouts)
	return layouts
}

func IsValidLayout(layout LayoutType) bool {
	for _, candidate := range validLayouts {
		if layout == candidate {
			return true
		}
	}
	return false
}

func (l LayoutType) PaneCount() int {
	switch l {
	case LayoutAgent:
		return 1
	case LayoutAgentShell:
		return 2
	case LayoutAgentShellLogs:
		return 3
	default:
		return 1
	}
}
```

- **agent** (1 pane): just the AI agent
- **agent-shell** (2 panes): agent + a shell for manual work
- **agent-shell-logs** (3 panes): agent + shell + a logs pane

### Storage (storage.go)

The `Store` uses mutex-protected JSON file I/O with atomic writes (write to `.tmp`, then rename). It also handles legacy migration — the original format was a bare JSON array of workspaces; the new format is the full `State` object. It detects the old format by peeking at the first JSON token:

```bash
sed -n '115,142p' internal/workspace/storage.go
```

```output
func (s *Store) loadStateUnsafe() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("reading %s: %w", s.path, err)
	}

	if len(data) == 0 {
		return State{}, nil
	}

	switch firstJSONToken(data) {
	case '[':
		var workspaces []Workspace
		if err := json.Unmarshal(data, &workspaces); err != nil {
			return State{}, fmt.Errorf("parsing %s: %w", s.path, err)
		}
		return migrateLegacyWorkspaces(workspaces), nil
	default:
		var state State
		if err := json.Unmarshal(data, &state); err != nil {
			return State{}, fmt.Errorf("parsing %s: %w", s.path, err)
		}
		return state, nil
	}
}
```

If the first non-whitespace byte is `[`, it is the old format and gets migrated by synthesizing Repository and Checkout records from the workspace data using deterministic UUID5 hashes (so the same data always produces the same IDs).

### Manager — The Heart of Workspace Lifecycle (manager.go)

The `Manager` orchestrates workspace creation, deletion, switching, and broadcasting. It depends on two interfaces:
- `SessionCreator`: tmux operations (create session, kill, split, switch, send keys)
- `CheckoutLifecycle`: worktree operations (create, remove, merge) via the `worktrunk` package

#### Creating a Workspace

There are three creation modes, each with a different code path:

```bash
sed -n '582,642p' internal/workspace/manager.go
```

```output
func (m *Manager) createRuntime(ctx context.Context, title string, agentType agent.AgentType, projectPath string, branch string, baseBranch string, backend Backend, ownership CheckoutOwnership, experimentID string, layout LayoutType) (*Workspace, func(), error) {
	id := uuid.New().String()
	sessionName := m.prefixedSessionName(title)

	agentPaneID, err := m.sessions.CreateSession(ctx, sessionName, projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("creating session for %q: %w", title, err)
	}
	cleanup := func() {
		_ = m.sessions.KillSession(context.Background(), sessionName)
	}

	paneTargets := map[string]string{"agent": agentPaneID}
	if layout.PaneCount() >= 2 {
		paneID, err := m.sessions.SplitWindow(ctx, sessionName, true, projectPath)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("splitting window for shell pane: %w", err)
		}
		paneTargets["shell"] = paneID
	}
	if layout.PaneCount() >= 3 {
		paneID, err := m.sessions.SplitWindow(ctx, sessionName, false, projectPath)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("splitting window for logs pane: %w", err)
		}
		paneTargets["logs"] = paneID
	}

	def, ok := agent.Get(agentType)
	if !ok {
		cleanup()
		return nil, nil, fmt.Errorf("agent type %q is not registered", agentType)
	}
	launchCmd := def.Binary
	for _, flag := range def.LaunchFlags {
		launchCmd += " " + flag
	}
	if err := m.sessions.SendKeys(ctx, agentPaneID, launchCmd, tmux.SendOptions{}); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("launching agent %q: %w", agentType, err)
	}

	return &Workspace{
		ID:                id,
		Title:             title,
		AgentType:         agentType,
		ProjectPath:       projectPath,
		Branch:            branch,
		BaseBranch:        baseBranch,
		CheckoutBackend:   backend,
		CheckoutOwnership: ownership,
		ExperimentID:      experimentID,
		Layout:            layout,
		Status:            agent.StatusIdle,
		SessionName:       sessionName,
		PaneTargets:       paneTargets,
		CreatedAt:         time.Now(),
	}, cleanup, nil
}
```

`createRuntime` is the shared core for all three creation modes. The sequence:

1. Generate a UUID and a prefixed session name (`colo-<title>`)
2. Create a detached tmux session in the project directory, capturing the first pane ID
3. Split panes according to the layout (horizontal split for shell, vertical for logs)
4. Look up the agent definition and send the launch command (e.g. `claude`) to the agent pane
5. Return the workspace record and a cleanup function (kills the session on failure)

The three creation modes wrap `createRuntime` differently:

- **CreateStandalone**: Uses `resolveStandaloneCheckout` (git inspection) to discover branch/repo info from an existing checkout, then creates the runtime
- **CreateWithWorktree**: Calls `worktrunk.Create` to spin up a new git worktree, then creates the runtime in that worktree path
- **CreateExperiment**: Creates N candidates (one per agent or N copies of one agent), each with its own worktree and workspace, optionally broadcasting a prompt to all of them

#### Deletion

Deletion is also carefully orchestrated:

```bash
sed -n '402,460p' internal/workspace/manager.go
```

```output
func (m *Manager) Delete(ctx context.Context, id string) error {
	state, err := m.store.LoadState()
	if err != nil {
		return fmt.Errorf("loading workspace state: %w", err)
	}

	ws, found := findWorkspace(state.Workspaces, id)
	if !found {
		return fmt.Errorf("workspace %q not found", id)
	}

	removeCheckout := false
	var checkoutRecord Checkout
	if ws.CheckoutID != "" {
		if record, ok := findCheckout(state.Checkouts, ws.CheckoutID); ok {
			checkoutRecord = record
			if ws.CheckoutOwnership == OwnershipColosseumManaged && countWorkspaceRefs(state.Workspaces, ws.CheckoutID) == 1 {
				removeCheckout = true
			}
		}
	}

	if removeCheckout {
		if err := m.removeManagedCheckout(ctx, checkoutRecord); err != nil {
			return fmt.Errorf("removing worktree %q: %w", checkoutRecord.Branch, err)
		}
	}

	if err := m.sessions.KillSession(ctx, m.workspaceSessionName(ws)); err != nil && !tmux.IsSessionNotFound(err) {
		return fmt.Errorf("killing session for %q: %w", ws.Title, err)
	}

	if err := m.store.UpdateState(func(next *State) error {
		next.Workspaces = filterWorkspaces(next.Workspaces, id)
		if removeCheckout {
			next.Checkouts = filterCheckouts(next.Checkouts, checkoutRecord.ID)
		}
		for i := range next.Experiments {
			next.Experiments[i].WorkspaceIDs = filterIDs(next.Experiments[i].WorkspaceIDs, id)
			if removeCheckout {
				next.Experiments[i].CheckoutIDs = filterIDs(next.Experiments[i].CheckoutIDs, checkoutRecord.ID)
				if next.Experiments[i].WinnerCheckoutID == checkoutRecord.ID {
					next.Experiments[i].WinnerCheckoutID = ""
				}
			}
			switch {
			case len(next.Experiments[i].CheckoutIDs) == 0:
				next.Experiments[i].Status = ExperimentAbandoned
			case len(next.Experiments[i].WorkspaceIDs) == 0 && next.Experiments[i].WinnerCheckoutID == "":
				next.Experiments[i].Status = ExperimentAwaitingEvaluation
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("removing workspace %q: %w", id, err)
	}

	return nil
}
```

Delete follows a careful cleanup order:

1. Check if this is the last workspace referencing a managed checkout — if so, remove the worktree
2. Kill the tmux session (tolerating "session not found" if it is already dead)
3. Update the state atomically: remove workspace, remove checkout if applicable, update experiment references (cascading status changes: no checkouts left = Abandoned, no workspaces left = AwaitingEvaluation)

#### Broadcasting

Broadcasting sends the same prompt text to the agent pane of multiple workspaces:

```bash
sed -n '490,557p' internal/workspace/manager.go
```

```output
func (m *Manager) Broadcast(ctx context.Context, prompt string, workspaceIDs []string) (BroadcastResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return BroadcastResult{}, fmt.Errorf("broadcast prompt cannot be empty")
	}
	if len(workspaceIDs) == 0 {
		return BroadcastResult{}, fmt.Errorf("broadcast requires at least one workspace")
	}

	workspaces, err := m.store.List()
	if err != nil {
		return BroadcastResult{}, fmt.Errorf("listing workspaces: %w", err)
	}

	byID := make(map[string]Workspace, len(workspaces))
	for _, ws := range workspaces {
		byID[ws.ID] = ws
	}

	seen := make(map[string]struct{}, len(workspaceIDs))
	result := BroadcastResult{}

	for _, id := range workspaceIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result.Requested++

		ws, ok := byID[id]
		if !ok {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID: id,
				Err:         fmt.Errorf("workspace not found"),
			})
			continue
		}

		agentPane := ws.PaneTargets["agent"]
		if agentPane == "" {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID:    ws.ID,
				WorkspaceTitle: ws.Title,
				Err:            fmt.Errorf("workspace has no agent pane"),
			})
			continue
		}

		opts := tmux.SendOptions{}
		if def, ok := agent.Get(ws.AgentType); ok {
			opts.InputDelay = def.InputDelay
			opts.ForcePaste = def.PasteSingleLine && !strings.Contains(prompt, "\n")
			opts.DisableBracketedPaste = def.DisableBracketedPasteForMultiline && strings.Contains(prompt, "\n")
		}

		if err := m.sessions.SendKeys(ctx, agentPane, prompt, opts); err != nil {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID:    ws.ID,
				WorkspaceTitle: ws.Title,
				Err:            fmt.Errorf("send prompt: %w", err),
			})
			continue
		}

		result.Delivered = append(result.Delivered, ws.Title)
	}

	return result, nil
}
```

Broadcast iterates over workspace IDs, deduplicating, and for each one looks up the agent pane target and the agent-specific `SendOptions` (input delay, paste mode, bracketed paste behavior). It tracks delivered vs. failed with `BroadcastResult`. This is the mechanism that powers "send the same task to 5 agents competing in an experiment."

## 7. Git Inspection — internal/workspace/git.go

The `GitInspector` interface wraps four git operations needed for standalone workspace creation:

```bash
sed -n '10,15p' internal/workspace/git.go
```

```output
type GitInspector interface {
	RepoRoot(ctx context.Context, path string) (string, error)
	CurrentBranch(ctx context.Context, path string) (string, error)
	DefaultBranch(ctx context.Context, path string) (string, error)
	MergeBase(ctx context.Context, path string, left string, right string) (string, error)
}
```

Each method is a thin wrapper around `git -C <path> <command>`. `DefaultBranch` first tries `symbolic-ref refs/remotes/origin/HEAD` (which gives the remote default branch), falling back to the current branch. These are used by `resolveStandaloneCheckout` to populate the workspace Snapshot without requiring the `wt` tool.

## 8. Worktrunk Client — internal/worktrunk

For managed worktrees, Colosseum delegates to an external `wt` CLI tool via the `worktrunk` package:

```bash
grep -n 'func (c \*Client)' internal/worktrunk/client.go
```

```output
107:func (c *Client) IsAvailable() bool {
112:func (c *Client) Create(ctx context.Context, repoPath, branch, base string) (Snapshot, error) {
124:func (c *Client) Remove(ctx context.Context, repoPath string, branches ...string) error {
131:func (c *Client) Merge(ctx context.Context, checkoutPath, target string) error {
140:func (c *Client) CopyIgnored(ctx context.Context, checkoutPath string) error {
145:func (c *Client) List(ctx context.Context, repoPath string) ([]Info, error) {
157:func (c *Client) ResolvePath(ctx context.Context, checkoutPath string) (Snapshot, error) {
208:func (c *Client) ResolveBranch(ctx context.Context, repoPath, branch, base string) (Snapshot, error) {
244:func (c *Client) defaultBranch(ctx context.Context, repoPath string) (string, error) {
269:func (c *Client) mergeBase(ctx context.Context, repoPath, branch, base string) (string, error) {
```

The worktrunk client calls out to `wt` (via `os/exec`) for worktree management:

- `IsAvailable()`: checks if `wt` is in PATH
- `Create()`: runs `wt switch --create --no-cd` to create a new worktree
- `Remove()`: runs `wt remove --foreground --yes --no-delete-branch`
- `List()`: parses JSON output from `wt list --format=json`
- `ResolvePath()` / `ResolveBranch()`: combine `wt list` with git merge-base to build a full `Snapshot`

The `Snapshot` struct captures all the metadata about a checkout that the workspace manager needs:
- RepoRoot, CheckoutPath, Branch, BaseBranch, DefaultBranch, MergeBaseSHA

## 9. Status Detection Engine — internal/status

This is the most intricate part of the system. The status engine continuously monitors what each AI agent is doing by capturing tmux pane content and matching it against regex patterns.

### ANSI Stripping (normalize.go)

Before any pattern matching, raw terminal output is cleaned:

```bash
cat internal/status/normalize.go
```

```output
package status

import "strings"

// StripANSI removes ANSI escape sequences from terminal output using a
// single-pass O(n) parser. This prevents escape codes embedded in pane
// content from interfering with regex-based status detection.
func StripANSI(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '\x1b' {
			buf.WriteByte(s[i])
			i++
			continue
		}

		// ESC found — determine sequence type
		i++
		if i >= len(s) {
			break
		}

		switch s[i] {
		case '[': // CSI: ESC [ <params> <final byte 0x40-0x7E>
			i++
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7E) {
				i++
			}
			if i < len(s) {
				i++ // skip final byte
			}

		case ']': // OSC: ESC ] <text> (BEL | ST)
			i++
			for i < len(s) {
				if s[i] == '\x07' {
					i++
					break
				}
				if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}

		default: // Simple two-byte escape (e.g. ESC M, ESC 7)
			if s[i] >= 0x40 && s[i] <= 0x5F {
				i++
			}
		}
	}

	return buf.String()
}
```

A single-pass O(n) parser that handles three escape sequence types:
- **CSI** (Control Sequence Introducer): `ESC[` followed by parameters and a final byte — this covers color codes, cursor movement, etc.
- **OSC** (Operating System Command): `ESC]` followed by text terminated by BEL or ST — this covers title setting, hyperlinks, etc.
- **Simple escapes**: two-byte sequences like `ESC M`

### Detector (detector.go)

The `Detector` is the core status classification engine:

```bash
sed -n '38,122p' internal/status/detector.go
```

```output
func (d *Detector) Detect(ctx context.Context, paneTarget string, agentType agent.AgentType) (agent.Status, string, error) {
	content, err := d.capturer.CapturePane(ctx, paneTarget, d.captureLines)
	if err != nil {
		return agent.StatusStopped, "", fmt.Errorf("capture pane %q: %w", paneTarget, err)
	}

	def, ok := agent.Get(agentType)
	if !ok {
		return agent.StatusUnknown, content, fmt.Errorf("unknown agent type: %s", agentType)
	}

	status := DetectFromContent(content, def)

	// Pane title is a supplementary working signal: Claude Code sets braille
	// spinner characters in the tmux pane title while actively processing.
	// Only upgrade Unknown — when content definitively says Idle we trust it
	// because the pane title is sticky and may not be cleared after a crash.
	if status == agent.StatusUnknown {
		if title, err := d.capturer.CapturePaneTitle(ctx, paneTarget); err == nil && titleIndicatesWorking(title) {
			status = agent.StatusWorking
		}
	}

	return status, content, nil
}

func DetectFromContent(content string, def *agent.AgentDef) agent.Status {
	content = StripANSI(content)
	lines := strings.Split(content, "\n")

	// Filter out lines matching agent-specific ignore patterns (e.g. status bars)
	// before running detection, so UI chrome doesn't trigger false matches.
	filtered := filterIgnored(lines, def.IgnorePatterns)

	lastNonEmpty := lastNonEmptyLines(filtered, 30)

	// Check only the very last non-empty line for idle indicators.
	// A prompt at the bottom means the agent is idle, regardless of
	// keywords in earlier output. Using only the last line avoids
	// false idle detection from selection arrows (❯ Yes, allow once)
	// which appear above other menu items in waiting prompts.
	var bottom []string
	if len(lastNonEmpty) > 0 {
		bottom = lastNonEmpty[len(lastNonEmpty)-1:]
	}
	if matchesAnyLine(bottom, def.IdlePatterns) {
		recent := linesBeforeBottom(lastNonEmpty, 3)

		// If the agent is visibly still running in the lines above the
		// prompt, keep it in Working. Check 5 lines because Claude's
		// thinking indicators can be several lines above the prompt
		// (separated by progress bars and decorative lines).
		if matchesAny(linesBeforeBottom(lastNonEmpty, 5), def.WorkingPatterns) {
			return agent.StatusWorking
		}

		// A visible prompt with a recent explicit question/choice means the
		// agent is waiting on the user, not merely idle at a fresh prompt.
		if isPromptOnly(bottom[0]) {
			if matchesAny(recent, def.WaitingPatterns) || matchesApprovalQuestion(recent) {
				return agent.StatusWaiting
			}
		}

		return agent.StatusIdle
	}

	if matchesAny(lastNonEmpty, def.WorkingPatterns) {
		return agent.StatusWorking
	}
	// Waiting patterns are more prone to false positives from old output,
	// so restrict to the last 10 non-empty lines in the non-idle path.
	recentForWaiting := lastNonEmpty
	if len(recentForWaiting) > 10 {
		recentForWaiting = recentForWaiting[len(recentForWaiting)-10:]
	}
	if matchesAny(recentForWaiting, def.WaitingPatterns) {
		return agent.StatusWaiting
	}
	if matchesAny(lastNonEmpty, def.ErrorPatterns) {
		return agent.StatusError
	}

	return agent.StatusUnknown
}
```

The detection algorithm is nuanced, designed to avoid false positives:

1. Strip ANSI codes and filter out agent-specific noise lines (status bars, decorative lines)
2. Extract the last 30 non-empty lines for analysis
3. **Idle check first** (only the very last line): if the bottom line is a bare prompt (`>`, `$`, `❯`), check whether the agent is *actually* idle or just showing a prompt while still working:
   - If working patterns appear 5 lines above the prompt → still Working
   - If waiting/approval patterns appear 3 lines above a bare prompt → Waiting (agent asked a question, prompt is just the input area)
   - Otherwise → truly Idle
4. **Working check**: any working pattern in the last 30 lines
5. **Waiting check**: restricted to last 10 lines (to avoid stale approval prompts from scrollback)
6. **Error check**: any error pattern in the last 30 lines
7. **Unknown fallback**: if none match, check the pane *title* for braille spinner characters (Claude Code sets these) — if found, upgrade to Working

### Poller (poller.go)

The Poller runs in a background goroutine and applies anti-flicker filtering:

```bash
sed -n '166,199p' internal/status/poller.go
```

```output
// shouldTransition applies spike detection and hysteresis filtering to
// prevent status flicker from transient terminal content (spinner
// animations, dynamic counters). Urgent statuses (Waiting, Error,
// Stopped) and initial detection bypass filtering entirely.
func (p *Poller) shouldTransition(state *workspaceState, detected agent.Status, now time.Time) bool {
	if detected == state.confirmed {
		state.pending = 0
		state.pendingFirst = time.Time{}
		return false
	}

	// Immediate transitions: urgent states and first detection.
	if isUrgentStatus(detected) || state.confirmed == agent.StatusUnknown {
		return true
	}

	// Track the candidate state.
	if detected != state.pending {
		state.pending = detected
		state.pendingFirst = now
	}

	// Spike window: new state must be sustained.
	if p.spikeWindow > 0 && now.Sub(state.pendingFirst) < p.spikeWindow {
		return false
	}

	// Hysteresis: current state must have been held long enough.
	if p.hysteresisWindow > 0 && now.Sub(state.confirmedAt) < p.hysteresisWindow {
		return false
	}

	return true
}
```

The poller maintains per-workspace state with two time windows:

- **Spike window** (1s): a detected status must persist for at least 1 second before being accepted, preventing spinner frame flicker from causing rapid Working→Idle→Working transitions
- **Hysteresis window** (500ms): the current confirmed status must have been held for at least 500ms before accepting a new one, preventing oscillation at the boundary

Urgent statuses (Waiting, Error, Stopped) bypass both filters for instant response — you want to know immediately when an agent asks a question or hits an error.

The poller sends `Update` messages through a buffered channel (capacity 64) that the TUI consumes.

## 10. The TUI — internal/tui

The TUI is built with Bubble Tea (the Elm Architecture for Go terminals). The main `App` model composes four sub-models:

### App Architecture (app.go)

```bash
sed -n '24,71p' internal/tui/app.go
```

```output
const (
	viewNormal viewState = iota
	viewNewWorkspace
	viewDeleteConfirm
	viewBroadcast
	viewHelp
)

type StatusUpdateMsg status.Update

type errMsg struct{ err error }

type workspaceCreatedMsg struct{ ws *workspace.Workspace }
type experimentCreatedMsg struct {
	result *workspace.ExperimentCreateResult
}

type workspaceDeletedMsg struct{ id string }
type broadcastCompletedMsg struct{ result workspace.BroadcastResult }
type previewRefreshMsg time.Time

var paneOrder = []string{"agent", "shell", "logs"}

type App struct {
	state                  viewState
	sidebar                sidebar.Model
	preview                preview.Model
	newDialog              dialog.NewWorkspaceModel
	delDialog              dialog.DeleteModel
	broadcastDialog        dialog.BroadcastModel
	helpDialog             dialog.HelpModel
	keys                   KeyMap
	keyConfig              config.KeysConfig
	returnKey              string
	theme                  theme.Theme
	store                  *workspace.Store
	manager                *workspace.Manager
	poller                 *status.Poller
	detector               *status.Detector
	previewRefreshInterval time.Duration
	sidebarMinWidth        int
	sidebarMaxWidth        int
	focusedPaneIdx         int
	statusBar              string
	width                  int
	height                 int
	ready                  bool
}
```

The App has five view states: Normal (the main dashboard), and four overlay dialogs (New Workspace, Delete Confirm, Broadcast, Help). The `Update` method dispatches based on view state — global messages (window resize, status updates, async results) are handled first, then delegate to the appropriate sub-handler.

### Initialization

On startup, three concurrent processes kick off:

```bash
sed -n '93,99p' internal/tui/app.go
```

```output
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.loadWorkspaces,
		a.listenForUpdates(),
		a.schedulePreviewRefresh(),
	)
}
```

1. `loadWorkspaces`: reads the JSON state, then runs a one-shot status refresh against all workspaces (saving back if any changed)
2. `listenForUpdates`: blocks on the poller channel, converting each `status.Update` into a `StatusUpdateMsg` for the TUI
3. `schedulePreviewRefresh`: sets a periodic timer (750ms default) to re-capture the selected panes terminal output

### The Dashboard View

The dashboard layout is sidebar + preview panel, joined horizontally:

```bash
sed -n '380,412p' internal/tui/app.go
```

```output
func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	sidebarView := a.sidebar.View()
	previewView := a.preview.View()
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, previewView)

	helpBar := renderShortHelp(a.theme, a.keys)
	if a.statusBar != "" {
		helpBar = a.theme.StatusWaiting.Render(a.statusBar) + "  " + helpBar
	}

	view := lipgloss.JoinVertical(lipgloss.Left, main, helpBar)

	switch a.state {
	case viewNewWorkspace:
		overlay := a.newDialog.View()
		view = a.placeOverlay(view, overlay)
	case viewDeleteConfirm:
		overlay := a.delDialog.View()
		view = a.placeOverlay(view, overlay)
	case viewBroadcast:
		overlay := a.broadcastDialog.View()
		view = a.placeOverlay(view, overlay)
	case viewHelp:
		overlay := a.helpDialog.View()
		view = a.placeOverlay(view, overlay)
	}

	return view
}
```

The rendering pipeline:
1. Render sidebar (workspace list with status icons) and preview (terminal content viewport)
2. Join them horizontally
3. Add a bottom help bar with key hints (and any status message)
4. If a dialog is active, center it over the main view using `lipgloss.Place`

### Sidebar (sidebar/)

The sidebar renders each workspace as a two-line item:

```bash
sed -n '25,53p' internal/tui/sidebar/view.go
```

```output
	for i, ws := range m.Workspaces {
		icon := theme.StatusIcon(ws.Status)
		styledIcon := t.StatusStyle(ws.Status).Render(icon)

		title := ws.Title
		agentStr := t.AgentName.Render(string(ws.AgentType))
		statusStr := t.StatusStyle(ws.Status).Render(ws.Status.String())
		branchStr := ""
		if ws.Branch != "" {
			branchStr = t.BranchName.Render(fmt.Sprintf("[%s]", ws.Branch))
		}

		line1 := fmt.Sprintf("  %s %s %s", styledIcon, title, branchStr)
		line2 := fmt.Sprintf("    %s · %s", agentStr, statusStr)

		if i == m.Cursor {
			line1 = t.SelectedItem.Width(m.Width - 2).Render(line1)
			line2 = t.SelectedItem.Width(m.Width - 2).Render(line2)
		}

		b.WriteString(line1)
		b.WriteString("\n")
		b.WriteString(line2)
		b.WriteString("\n")

		if i < len(m.Workspaces)-1 {
			b.WriteString("\n")
		}
	}
```

Each workspace shows:
- Line 1: status icon (color-coded) + title + branch name (if any)
- Line 2: agent type + status text

Status icons are: ● Working (green), ◉ Waiting (yellow), ○ Idle (gray), ■ Stopped (dark gray), ✗ Error (red)

The selected item gets a highlighted background. Navigation (j/k) updates the cursor, which triggers a preview refresh.

### Preview (preview/)

The preview panel wraps a Bubble Tea viewport for scrollable terminal output. It supports tabs for workspaces with multiple panes — h/l cycles between "agent", "shell", and "logs" tabs. The `updatePreviewContent` method in app.go captures the focused panes content via the Detector and sets it as the viewport content. Content is word-wrapped to fit the viewport width.

### Dialogs (dialog/)

Four overlay dialogs handle user interactions:

1. **NewWorkspaceModel**: Multi-field form (title, path with tab-completion, branch, mode selector, agent/layout pickers). Adapts its visible fields based on the create mode — experiment mode shows prompt input and agent strategy; worktree mode shows branch fields.
2. **DeleteModel**: Simple y/n confirmation that mentions whether a managed worktree will be removed.
3. **BroadcastModel**: Two-panel dialog with a target selector (toggle workspaces on/off) and a textarea for the prompt.
4. **HelpModel**: Lists all keybindings, split into available and unavailable features.

### Theme (theme/)

All colors and styles flow from the TOML config through `ThemeFromConfig`, which creates lipgloss styles for every UI element. The theme is threaded through all sub-models via `WithTheme()` builder methods.

## 11. The Dashboard Command — Tying It All Together

```bash
cat cmd/colosseum/dashboard.go
```

```output
package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tui"
)

func runDashboard(_ *cobra.Command, _ []string) error {
	store := newStore()
	client := newTmuxClient()
	mgr := newManager(store, client)

	detector := status.NewDetector(client, cfg.Status.CaptureLines)
	poller := status.NewPoller(detector, store, time.Duration(cfg.Status.PollIntervalMS)*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go poller.Run(ctx)

	app := tui.NewApp(store, mgr, poller, detector, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := p.Run()
	return err
}
```

This is where everything comes together. `runDashboard`:

1. Creates the Store (JSON file), tmux Client, and workspace Manager
2. Creates the Detector (captures panes, matches patterns) and Poller (background goroutine)
3. Starts the poller in a goroutine with a cancellable context
4. Creates the TUI App with all dependencies injected
5. Runs the Bubble Tea program in alt-screen mode with mouse support

The context cancellation ensures the poller goroutine stops when the TUI exits.

## 12. CLI Subcommands

Besides the TUI dashboard, Colosseum provides headless CLI commands for scripting:

- **`colosseum new <name>`**: Creates a workspace via one of three modes (existing checkout, new worktree, experiment). Supports flags for path, agent, branch, layout, and experiment options.
- **`colosseum list`**: Lists all workspaces with a one-shot status refresh, printing status icons.
- **`colosseum attach <name>`**: Switches the tmux client to the named workspace.
- **`colosseum delete <name>`**: Deletes a workspace (same cleanup logic as the TUI).
- **`colosseum broadcast --prompt "..." --workspaces "a,b,c"`**: Sends a prompt to multiple workspaces by name.

## 13. Data Flow Summary

Here is the complete data flow for the main use case — monitoring agents:

```
tmux pane content
    → CapturePane (tmux capture-pane -p)
    → StripANSI (O(n) parser)
    → filterIgnored (agent-specific noise removal)
    → DetectFromContent (last-line idle check, then working/waiting/error patterns)
    → shouldTransition (spike + hysteresis filtering)
    → Update channel (buffered, capacity 64)
    → StatusUpdateMsg (Bubble Tea message)
    → sidebar.UpdateWorkspaceStatus (updates icon + color)
    → preview.SetContent (refreshes terminal view)
```

And for workspace creation:

```
CLI flag / TUI dialog
    → validateCreate (title uniqueness, agent support, layout check)
    → resolveStandaloneCheckout / worktrunk.Create / experimentAgents
    → createRuntime:
        → tmux new-session → pane ID
        → tmux split-window (if layout needs shell/logs panes)
        → agent.Get → SendKeys "claude" (or codex, opencode, etc.)
    → store.UpdateState → atomic JSON write
```

## 14. Testing Strategy

The project is testable because of its interface-driven design. Let us verify the test suite passes:

```bash
go test ./... 2>&1 | tail -20
```

```output
ok  	github.com/ramtinj/colosseum/cmd/colosseum	(cached)
ok  	github.com/ramtinj/colosseum/internal/agent	(cached)
ok  	github.com/ramtinj/colosseum/internal/config	(cached)
ok  	github.com/ramtinj/colosseum/internal/status	(cached)
ok  	github.com/ramtinj/colosseum/internal/tmux	(cached)
ok  	github.com/ramtinj/colosseum/internal/tui	0.768s
ok  	github.com/ramtinj/colosseum/internal/tui/dialog	0.008s
ok  	github.com/ramtinj/colosseum/internal/tui/preview	0.005s
ok  	github.com/ramtinj/colosseum/internal/tui/sidebar	0.005s
?   	github.com/ramtinj/colosseum/internal/tui/theme	[no test files]
ok  	github.com/ramtinj/colosseum/internal/workspace	(cached)
ok  	github.com/ramtinj/colosseum/internal/worktrunk	(cached)
```

All packages pass. Key testing patterns:

- **tmux/**: `MockCommander` provides pre-programmed responses; tests verify exact tmux command sequences
- **status/**: `DetectFromContent` is tested with raw terminal content strings, verifying pattern matching without needing tmux. The poller uses configurable spike/hysteresis windows for deterministic timing tests.
- **workspace/**: Tests use temp directories for the JSON store and mock session creators
- **tui/**: Bubble Tea models are tested by sending `tea.KeyMsg` and `tea.WindowSizeMsg` messages and checking state transitions
- **worktrunk/**: Mock commanders simulate `wt` and `git` CLI responses

## 15. Key Design Decisions

1. **No tmux library — pure os/exec**: The `Commander` interface wraps raw `tmux` shell commands. This avoids C library dependencies, keeps the binary statically linkable, and makes the tmux interaction fully transparent/debuggable.

2. **Regex-based status detection over API polling**: AI agents do not expose status APIs. Colosseum captures terminal content and pattern-matches it, which is fragile but universal — it works with any CLI tool.

3. **Spike + hysteresis filtering**: Terminal output is noisy (spinners, progress bars). The two-window approach prevents status flicker without sacrificing responsiveness for urgent transitions.

4. **Interfaces at the consumer, not the provider**: `PaneCapturer` is defined in `status/` (where it is used), not in `tmux/` (where it is implemented). This follows the Go idiom of small, consumer-defined interfaces.

5. **Atomic JSON writes**: Write to `.tmp`, then `os.Rename`. This prevents corrupted state files from partial writes or crashes.

6. **Experiments as first-class entities**: Rather than just managing individual workspaces, the data model supports grouping them into experiments with evaluation tracking, enabling "which agent does this best?" workflows.
