# Colosseum - AI Agent Workspace Manager

A terminal-agnostic TUI for managing parallel AI coding agents across git worktrees, built on tmux and Go.

---

## Vision

Colosseum manages multiple AI coding agents (Claude Code, Codex, Gemini, OpenCode, Aider) running in parallel across isolated git worktrees. Each agent gets its own tmux session with a configurable pane layout, and a central TUI dashboard provides real-time status, notifications, and workspace switching.

The key differentiator: **broadcast the same prompt to multiple agents in different worktrees**, then monitor their progress from a single dashboard with notification badges that tell you exactly which agent needs attention.

---

## Architecture

```
┌─ Any Terminal (kitty, alacritty, ghostty, foot, wezterm, ...) ────────┐
│                                                                        │
│  ┌─ tmux session: "colo-auth-feature" ───────────────────────────────┐ │
│  │ ┌─ %0 agent ────────────┬─ %1 shell ─────────────────────────┐   │ │
│  │ │ $ claude               │ $ npm run dev                      │   │ │
│  │ │ (esc to interrupt)     │ ready on :3000                     │   │ │
│  │ └───────────────────────┴─────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                        │
│  ┌─ tmux session: "colo-ctrl" (the TUI dashboard) ───────────────────┐ │
│  │ ┌─ Sidebar ────────────────────┬─ Preview ────────────────────┐   │ │
│  │ │                              │                               │   │ │
│  │ │  WORKSPACES                  │ auth-feature (claude)         │   │ │
│  │ │                              │ branch: main                  │   │ │
│  │ │  ● auth-feature    [main]    │ path: ~/proj                  │   │ │
│  │ │    claude · working          │                               │   │ │
│  │ │                              │ > Implementing JWT login...   │   │ │
│  │ │  ◉ api-v2       [feat/api]   │   Reading src/auth/handler   │   │ │
│  │ │    codex · waiting           │   (esc to interrupt)          │   │ │
│  │ │                              │                               │   │ │
│  │ │  ○ fix-tests    [bugfix]     │                               │   │ │
│  │ │    gemini · idle             │                               │   │ │
│  │ │                              │                               │   │ │
│  │ │  ■ refactor     [refactor]   │                               │   │ │
│  │ │    claude · stopped          │                               │   │ │
│  │ │                              │                               │   │ │
│  │ │  NOTIFICATIONS (2)           │                               │   │ │
│  │ │  ! api-v2: permission needed │                               │   │ │
│  │ │  ✓ fix-tests: task complete  │                               │   │ │
│  │ │                              │                               │   │ │
│  │ └──────────────────────────────┴───────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

### Core Concepts

- **Workspace**: A named unit of work. Maps 1:1 to a tmux session. Contains an agent pane, optionally a shell pane and log pane. Associated with a git worktree, branch, and agent type.
- **Agent**: An AI coding CLI (Claude Code, Codex, Gemini, etc.). Each agent type has its own detection patterns and launch flags.
- **Status**: The detected state of an agent: Working, Waiting, Idle, Stopped, or Error.
- **Notification**: A state-transition event (e.g., agent went from Working to Waiting). Stored with unread tracking per workspace.
- **Broadcast**: Sending the same prompt to multiple workspaces simultaneously.

### Session Model

The TUI runs in its own tmux session (`colo-ctrl`). Each workspace is a separate tmux session (`colo-{title}`). Switching to a workspace uses `tmux switch-client`. Returning to the dashboard uses the same mechanism.

This means:
- Workspaces are fully isolated (own environment, own panes, own scrollback)
- The TUI doesn't share screen real estate with agent panes
- SSH/headless workflows work naturally (attach to any session)
- Workspace pane layouts persist independently

---

## Features

### Workspace Management
- Create workspaces with a specific agent, project path, and optional git worktree branch
- Configurable pane layouts per workspace:
  - `agent` — single pane running the agent
  - `agent-shell` — agent left, shell right (for dev servers, manual commands)
  - `agent-shell-logs` — agent left, shell + logs stacked right
- Rename, delete, and reorder workspaces
- Workspace persistence across restarts (JSON state file)

### Agent Support
- Pluggable agent registry with per-agent configuration:
  - Binary name and detection method
  - Launch flags (including YOLO/auto-approve mode)
  - Custom instruction injection (Claude: `--append-system-prompt`)
  - Status detection patterns (unique per agent)
- Supported agents at launch:
  - Claude Code
  - Codex CLI
  - Gemini CLI
  - OpenCode
  - Aider

### Status Detection
- Background goroutine polls `tmux capture-pane` for each workspace (every 1-2 seconds)
- Per-agent regex pattern matching on the last 50 lines of pane output
- Detection signals:
  - **Working**: `(esc to interrupt)`, braille spinner chars (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`), activity indicators
  - **Waiting**: Permission prompts (`Yes, allow once`, `Allow always`), questions (`?` at EOL), selection menus (`❯ 1.`)
  - **Idle**: Prompt chars (`>`, `$`, `❯`, `╰─>`) at end of last non-empty line + no recent output
  - **Error**: Rate limit patterns (`429`), crash/panic text, auth failures
  - **Stopped**: tmux session doesn't exist
- Fixture-driven testing: real captured pane content stored in `testdata/fixtures/` and tested against detection functions

### Git Worktree Integration
- Create a git worktree per workspace, allowing multiple agents to work on the same repo in parallel on different branches
- Configurable worktree path templates (`../{repo}-worktrees/{branch}`)
- Broadcast a prompt to N workspaces: each agent receives the same task but works in its own isolated worktree
- Cleanup on workspace deletion: remove worktree directory and optionally delete the branch
- Diff viewer: compare worktree branch against base branch (main/master)

### Notification System
- In-memory notification store with per-workspace unread counts
- Notifications generated on status transitions:
  - Working → Waiting: "Permission needed" or "Question asked"
  - Working → Idle: "Task complete"
  - Working → Error: "Agent error" (with error type)
  - Any → Stopped: "Agent exited"
- Desktop notifications via `notify-send` (Linux)
- Claude Code hook integration: listens for `idle_prompt` and `permission_prompt` hooks for more reliable notification than scraping
- Jump-to-unread: `J` key navigates to the next workspace with unread notifications

### Broadcast Prompt
- Dialog to compose a prompt and select target workspaces (multi-select)
- Auto-create worktrees if needed (one branch per workspace, branching from the same base)
- Sends the prompt via `tmux send-keys` to each workspace's agent pane
- Ideal workflow: "implement feature X" → broadcast to Claude, Codex, and Gemini on separate worktrees → compare results with the diff viewer

### Diff Viewer
- Full-screen overlay triggered by `D` on a workspace
- Computes diff between the workspace's worktree branch and a configurable base branch
- Side-by-side rendering with additions (green) and deletions (red)
- File list panel with navigation
- Scroll through hunks with configurable context lines

### TUI Dashboard
- Bubble Tea application with two main panels:
  - **Sidebar** (left): workspace list with status icons, agent type, branch name, unread badges. Below: notification list.
  - **Preview** (right): last N lines of the selected workspace's agent pane output, refreshed on each poll cycle.
- Status icons:
  - `●` Working (green, animated)
  - `◉` Waiting (yellow, needs attention)
  - `○` Idle (dim)
  - `■` Stopped (gray)
  - `✗` Error (red)
- Lipgloss styling with theme support

### CLI
- `colosseum` — launch the TUI dashboard
- `colosseum new <name> --path <dir> --agent <type> [--branch <name>]` — create a workspace
- `colosseum list` — list workspaces and their statuses
- `colosseum attach <name>` — switch to a workspace's tmux session
- `colosseum broadcast --prompt <text> --workspaces <w1,w2,...>` — broadcast a prompt
- `colosseum delete <name> [--cleanup-worktree] [--delete-branch]` — delete a workspace
- `colosseum diff <name> [--base <branch>]` — show diff for a workspace

---

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Navigate workspace list |
| `Enter` | Attach to selected workspace (switch tmux session) |
| `n` | New workspace dialog |
| `d` | Delete workspace (with cleanup options) |
| `b` | Broadcast prompt dialog |
| `D` | Diff viewer for selected workspace |
| `r` | Rename workspace |
| `/` | Filter/search workspaces |
| `Tab` | Cycle sidebar sections (workspaces / notifications) |
| `m` | Mark selected workspace notifications as read |
| `J` | Jump to next workspace needing attention (Waiting/Error) |
| `R` | Restart agent in selected workspace |
| `s` | Stop agent in selected workspace |
| `?` | Help overlay |
| `q` | Quit |

---

## Project Layout

```
colosseum/
├── cmd/
│   └── colosseum/
│       └── main.go                  # Entry point, cobra CLI root command
│
├── internal/
│   ├── agent/                       # Agent type definitions and detection
│   │   ├── agent.go                 #   AgentDef struct, Status enum, AgentType constants
│   │   ├── registry.go              #   Global registry, lookup by name/alias, available agents
│   │   ├── claude.go                #   Claude Code: binary, flags, yolo mode, detection patterns
│   │   ├── codex.go                 #   Codex CLI: binary, flags, detection patterns
│   │   ├── gemini.go                #   Gemini CLI: binary, flags, detection patterns
│   │   ├── opencode.go              #   OpenCode: binary, flags, detection patterns
│   │   ├── aider.go                 #   Aider: binary, flags, detection patterns
│   │   └── patterns.go              #   Shared regex patterns (spinners, prompt chars, errors)
│   │
│   ├── status/                      # Background status detection engine
│   │   ├── types.go                 #   Status enum: Working, Waiting, Idle, Stopped, Error
│   │   ├── detector.go              #   Detector: captures pane content, dispatches to agent-specific patterns
│   │   └── poller.go                #   Background goroutine: polls all workspaces, sends updates via channel
│   │
│   ├── tmux/                        # tmux command abstraction layer
│   │   ├── client.go                #   Exec wrapper with error handling, timeout, session cache
│   │   ├── session.go               #   Session CRUD: create, kill, exists, attach, switch-client, list
│   │   ├── pane.go                  #   Pane ops: split, resize, capture-pane, send-keys, select-pane
│   │   └── format.go                #   Format string helpers for tmux format variables
│   │
│   ├── workspace/                   # Core domain model
│   │   ├── workspace.go             #   Workspace struct: ID, title, agent, worktree, status, panes, unread
│   │   ├── manager.go               #   Workspace lifecycle: create, delete, list, switch, broadcast
│   │   ├── layout.go                #   Pane layout templates and their tmux split commands
│   │   └── storage.go               #   JSON persistence to ~/.config/colosseum/workspaces.json
│   │
│   ├── worktree/                    # Git worktree management
│   │   ├── worktree.go              #   Create, remove, list worktrees via git CLI
│   │   ├── template.go              #   Path template resolution ({repo}, {branch}, {id})
│   │   └── diff.go                  #   Compute diff between worktree branch and base branch
│   │
│   ├── notification/                # Notification system
│   │   ├── types.go                 #   Notification struct: ID, workspace, title, body, read, time
│   │   ├── store.go                 #   In-memory store with per-workspace unread counts
│   │   └── desktop.go               #   Desktop notifications via notify-send
│   │
│   ├── hook/                        # Agent hook integration
│   │   └── claude.go                #   Claude Code hook listener (idle_prompt, permission_prompt)
│   │
│   ├── config/                      # Configuration
│   │   ├── config.go                #   Config struct, TOML loading, validation
│   │   └── paths.go                 #   XDG paths: config dir, data dir, state dir
│   │
│   └── tui/                         # Bubble Tea TUI
│       ├── app.go                   #   Root model: Init, Update, View, delegates to sub-models
│       ├── keys.go                  #   Global keymap definitions using key.Binding
│       │
│       ├── sidebar/                 #   Left panel: workspace list + notifications
│       │   ├── model.go             #     Bubble Tea model: cursor, selection, section toggle
│       │   ├── view.go              #     Lipgloss rendering: status icons, branch, badges
│       │   └── keys.go              #     Sidebar-local key bindings
│       │
│       ├── preview/                 #   Right panel: agent output preview
│       │   ├── model.go             #     Viewport wrapping captured pane content
│       │   └── view.go              #     Renders last N lines with ANSI passthrough
│       │
│       ├── diff/                    #   Diff viewer (full-screen overlay)
│       │   ├── model.go             #     File list + diff content, scroll state
│       │   └── view.go              #     Side-by-side diff rendering with colors
│       │
│       ├── dialog/                  #   Modal dialogs
│       │   ├── new_workspace.go     #     Form: name, path, agent picker, branch, layout
│       │   ├── broadcast.go         #     Prompt input + multi-select workspace picker
│       │   ├── delete.go            #     Confirm deletion with worktree/branch cleanup toggles
│       │   └── help.go              #     Keybinding reference overlay
│       │
│       └── theme/                   #   Styling
│           └── theme.go             #     Lipgloss style definitions, theme switching
│
├── testdata/                        # Fixture-driven status detection tests
│   └── fixtures/
│       ├── claude/
│       │   ├── working/             #   Captured pane content when Claude is working
│       │   │   ├── reading_file.txt
│       │   │   ├── writing_code.txt
│       │   │   └── spinner.txt
│       │   ├── waiting/             #   Permission prompts, questions
│       │   │   ├── permission_prompt.txt
│       │   │   ├── yes_no_question.txt
│       │   │   └── selection_menu.txt
│       │   ├── idle/                #   At prompt, ready for input
│       │   │   ├── fresh_prompt.txt
│       │   │   └── after_task.txt
│       │   └── error/               #   Rate limits, crashes
│       │       ├── rate_limit.txt
│       │       └── auth_error.txt
│       ├── codex/
│       │   ├── working/
│       │   ├── waiting/
│       │   ├── idle/
│       │   └── error/
│       └── gemini/
│           ├── working/
│           ├── waiting/
│           ├── idle/
│           └── error/
│
├── go.mod
├── go.sum
├── Makefile                         # build, test, lint, install targets
├── CLAUDE.md                        # Project conventions for Claude Code
└── AGENTS.md                        # Agent team configuration
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/charmbracelet/bubbles` | Standard TUI components (list, textinput, viewport, table) |
| `github.com/spf13/cobra` | CLI command parsing |
| `github.com/pelletier/go-toml/v2` | Config file parsing |
| `github.com/sergi/go-diff` | Diff computation |
| `github.com/google/uuid` | Workspace ID generation |

No external tmux library needed. tmux interaction is via `os/exec` calling the `tmux` binary directly (standard Go approach, same as claude-dashboard and NTM).

---

## Configuration

Location: `~/.config/colosseum/config.toml`

```toml
[general]
default_agent = "claude"
default_layout = "agent-shell"

[worktree]
enabled = true
path_template = "../{repo}-worktrees/{branch}"
auto_cleanup = true
delete_branch_on_cleanup = false

[status]
poll_interval_ms = 1500
capture_lines = 50

[notification]
desktop_enabled = true
on_waiting = true
on_idle = true
on_error = true

[theme]
name = "catppuccin"
```

---

## Implementation Phases

### Phase 1: Foundation
- Go module setup, directory structure
- tmux client: session create/kill/exists/switch, pane split/capture/send-keys
- Workspace model: struct, JSON persistence, basic CRUD
- Minimal TUI: sidebar with hardcoded workspaces, no preview

### Phase 2: Core TUI
- Bubble Tea app with sidebar + preview panels
- New workspace dialog (name, path, agent picker)
- Workspace switching via tmux switch-client
- Status detection: background poller with Claude patterns
- Status icons in sidebar

### Phase 3: Worktrees + Broadcast
- Git worktree create/remove/list
- Worktree integration in new workspace dialog
- Broadcast prompt dialog and execution
- Multiple agent support (Codex, Gemini, OpenCode, Aider detection patterns)

### Phase 4: Notifications + Diff
- Notification store with unread tracking
- Desktop notifications via notify-send
- Notification section in sidebar
- Jump-to-unread navigation
- Diff viewer overlay

### Phase 5: Polish
- Claude Code hook integration (idle_prompt, permission_prompt)
- Theme support
- Workspace pane layout options (agent-only, agent-shell, agent-shell-logs)
- CLI subcommands (new, list, attach, broadcast, delete, diff)
- Configuration file support
- Help overlay
- Fixture-driven test suite for all agent detection patterns
