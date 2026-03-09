# Colosseum - AI Agent Workspace Manager

A terminal-agnostic TUI for managing parallel AI coding agents across git worktrees, built on tmux and Go.

---

## Vision

Colosseum manages multiple AI coding agents (Claude Code, Codex, Gemini, OpenCode, Aider) running in parallel across isolated git worktrees. Each agent gets its own tmux session with a configurable pane layout, and a central TUI dashboard provides real-time status and workspace switching.

The key differentiator: **broadcast the same prompt to multiple agents in different worktrees**, then monitor their progress from a single dashboard while comparing isolated candidates side by side.

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
│  │ └──────────────────────────────┴───────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

### Core Concepts

- **Workspace**: A named unit of work. Maps 1:1 to a tmux session. Contains an agent pane, optionally a shell pane and log pane. Associated with a git worktree, branch, and agent type.
- **Agent**: An AI coding CLI (Claude Code, Codex, Gemini, etc.). Each agent type has its own detection patterns and launch flags.
- **Status**: The detected state of an agent: Working, Waiting, Idle, Stopped, or Error.
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

Status legend: `[x]` shipped, `[ ]` still to implement, `[-]` intentionally removed from scope.

### Workspace Management
- [x] Create a workspace from an existing checkout path
- [x] Create a managed worktree-backed workspace through `worktrunk`
- [x] Create an experiment run that fans out into several sibling worktree-backed workspaces
- [x] Configurable pane layouts per workspace:
  - [x] `agent` — single pane running the agent
  - [x] `agent-shell` — agent left, shell right
  - [x] `agent-shell-logs` — agent left, shell + logs stacked right
- [ ] Rename workspaces
- [x] Delete workspaces
- [ ] Reorder workspaces
- [x] Workspace persistence across restarts (JSON state file)

### Agent Support
- [x] Pluggable agent registry with per-agent configuration
- [x] Binary name and detection method
- [x] Launch flags and YOLO-mode definitions on agent records
- [ ] YOLO mode surfaced in the TUI or CLI create flow
- [ ] Custom instruction injection
- [x] Status detection patterns per agent
- [x] Claude Code supported for new workspace creation
- [x] Codex CLI supported for new workspace creation
- [ ] Gemini CLI supported for new workspace creation
- [x] OpenCode supported for new workspace creation
- [ ] Aider supported for new workspace creation
- [x] Legacy Gemini/Aider detection compatibility for existing workspaces

### Status Detection
- [x] Background polling of `tmux capture-pane`
- [x] Per-agent regex pattern matching on recent pane output
- [x] `Working` detection
- [x] `Waiting` detection
- [x] `Idle` detection
- [x] `Error` detection
- [x] `Stopped` detection
- [x] ANSI stripping before matching
- [x] Pane-title signal for supplementary working detection
- [x] Spike/hysteresis filtering to reduce flicker
- [x] Fixture-driven testing across Claude, Codex, Gemini, OpenCode, and Aider samples

### Git Worktree Integration
- [x] Create a managed worktree per workspace
- [x] Delegate worktree path templates and placement to `worktrunk`
- [x] Ownership-aware cleanup on workspace deletion
- [x] Experiment-run creation of several sibling managed worktrees
- [x] Persist repository, checkout, experiment, and evaluation metadata for compare/vote readiness
- [ ] Generic broadcast flow that auto-creates worktrees on demand
- [ ] Diff viewer against a base branch

### Notifications
- [-] In-app notification center and unread tracking
- [-] Desktop notifications via `notify-send`
- [-] Notification-driven Claude hook integration
- [-] Jump-to-unread workflow
- [x] Jump-to-attention shortcut for waiting/error workspaces remains in scope

### Broadcast Prompt
- [x] Dialog to compose a prompt and select target workspaces
- [x] Multi-select target picker
- [x] Send the prompt via `tmux send-keys` to each selected workspace
- [x] CLI `broadcast` command
- [x] Optional prompt broadcast immediately after experiment creation
- [ ] Auto-create worktrees as part of the generic broadcast flow
- [ ] In-app compare/vote workflow after broadcast fan-out

### Diff Viewer
- [ ] Full-screen overlay triggered by `D`
- [ ] Diff computation against a configurable base branch
- [ ] Side-by-side rendering with additions/deletions
- [ ] File list panel with navigation
- [ ] Hunk scrolling with configurable context lines

### TUI Dashboard
- [x] Bubble Tea dashboard with sidebar + preview layout
- [x] Sidebar workspace list with status icons, agent type, and branch name
- [-] Sidebar notification list and unread badges
- [x] Preview of the selected workspace pane output
- [x] Status icons for Working / Waiting / Idle / Stopped / Error
- [x] Lipgloss styling with theme support
- [x] New workspace dialog
- [x] Broadcast dialog
- [x] Delete confirmation dialog
- [x] Help overlay
- [ ] Filter/search workspaces
- [ ] Repo-centric UI

### CLI
- [x] `colosseum` — launch the TUI dashboard
- [x] `colosseum new` — create an existing-checkout workspace, managed worktree workspace, or experiment run
- [x] `colosseum list`
- [x] `colosseum attach`
- [x] `colosseum broadcast --prompt <text> --workspaces <w1,w2,...>`
- [x] `colosseum delete`
- [ ] `colosseum diff`

---

## Keybindings

| Status | Key | Action |
|--------|-----|--------|
| `[x]` | `j` / `k` | Navigate workspace list |
| `[x]` | `Enter` | Attach to selected workspace |
| `[x]` | `n` | New workspace dialog |
| `[x]` | `d` | Delete workspace |
| `[x]` | `b` | Broadcast prompt dialog |
| `[ ]` | `D` | Diff viewer for selected workspace |
| `[ ]` | `r` | Rename workspace |
| `[ ]` | `/` | Filter/search workspaces |
| `[x]` | `Tab` | Cycle sidebar sections |
| `[-]` | `m` | Mark notifications as read |
| `[x]` | `J` | Jump to next workspace needing attention |
| `[ ]` | `R` | Restart agent in selected workspace |
| `[ ]` | `s` | Stop agent in selected workspace |
| `[x]` | `?` | Help overlay |
| `[x]` | `q` | Quit |

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

[theme]
name = "catppuccin"
```

---

## Implementation Phases

### Phase 1: Foundation
- [x] Go module setup and directory structure
- [x] tmux client: session create/kill/exists/switch, pane split/capture/send-keys
- [x] Workspace model and JSON persistence
- [x] Minimal TUI foundation

### Phase 2: Core TUI
- [x] Bubble Tea app with sidebar + preview panels
- [x] New workspace dialog
- [x] Workspace switching via `tmux switch-client`
- [x] Status detection background poller
- [x] Status icons in the sidebar

### Phase 3: Worktrees + Broadcast
- [x] Managed worktree create/remove/list through `worktrunk`
- [x] Worktree integration in the new workspace dialog
- [x] Broadcast prompt dialog and execution
- [x] Experiment-run fan-out creation
- [x] Multiple detection patterns for Codex, Gemini, OpenCode, and Aider
- [ ] Gemini workspace creation support
- [ ] Aider workspace creation support

### Phase 4: Notifications + Diff
- [-] Notification store with unread tracking
- [-] Desktop notifications via `notify-send`
- [-] Notification section in sidebar
- [-] Jump-to-unread navigation
- [ ] Diff viewer overlay

### Phase 5: Polish
- [ ] Claude Code hook integration
- [x] Theme support
- [x] Workspace pane layout options (`agent`, `agent-shell`, `agent-shell-logs`)
- [x] CLI subcommands (`new`, `list`, `attach`, `broadcast`, `delete`)
- [ ] CLI `diff` subcommand
- [x] Configuration file support
- [x] Help overlay
- [x] Fixture-driven test suite for all agent detection patterns
