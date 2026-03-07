# Colosseum

A terminal-agnostic TUI for managing parallel AI coding agents across git worktrees, built on tmux and Go.

```
┌─ tmux session: "dashboard" (the TUI you launched) ────────────────┐
│ ┌─ Sidebar ────────────────────┬─ Preview ────────────────────┐   │
│ │                              │                               │   │
│ │  WORKSPACES                  │ auth-feature (claude)         │   │
│ │                              │ branch: main                  │   │
│ │  ● auth-feature    [main]   │                               │   │
│ │    claude · working          │ > Implementing JWT login...   │   │
│ │                              │   Reading src/auth/handler    │   │
│ │  ◉ api-v2       [feat/api]  │   (esc to interrupt)          │   │
│ │    codex · waiting           │                               │   │
│ │                              │                               │   │
│ │  ○ fix-tests    [bugfix]    │                               │   │
│ │    gemini · idle             │                               │   │
│ │                              │                               │   │
│ └──────────────────────────────┴───────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
```

Colosseum currently supports Claude Code and Codex workspaces. Each agent gets its own tmux session with a configurable pane layout, and a central TUI dashboard provides real-time status detection and workspace switching.

## Prerequisites

- **Go 1.24+**
- **tmux 3.0+**

## Installation

### From source

```bash
git clone https://github.com/ramtinj/colosseum.git
cd colosseum
make build
```

This produces a `./colosseum` binary. Optionally install it to your `$GOPATH/bin`:

```bash
make install
```

### Go install

```bash
go install github.com/ramtinj/colosseum/cmd/colosseum@latest
```

## Usage

### Launch the dashboard

```bash
colosseum
```

Opens the Bubble Tea TUI with a sidebar listing all workspaces and a preview panel showing the selected workspace's agent output. Status is polled in the background every 1.5 seconds.

### Create a workspace

```bash
colosseum new my-feature --path ~/projects/myapp --agent claude --branch feat/auth --layout agent-shell
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--path` | `-p` | `.` | Project directory |
| `--agent` | `-a` | `claude` | Agent type: `claude`, `codex` |
| `--branch` | `-b` | | Git branch name |
| `--layout` | `-l` | `agent-shell` | Pane layout: `agent`, `agent-shell`, `agent-shell-logs` |

This creates a tmux session named `colo-my-feature` with panes arranged per the chosen layout.

### List workspaces

```bash
colosseum list
```

```
  ● auth-feature [main] (claude · Working)
  ◉ api-v2 [feat/api] (codex · Waiting)
  ○ fix-tests [bugfix] (codex · Idle)
```

### Attach to a workspace

```bash
colosseum attach my-feature
```

Switches your tmux client to the workspace's session. Colosseum also installs a tmux `prefix+e` binding that returns directly to the dashboard session you launched it from.

### Delete a workspace

```bash
colosseum delete my-feature
```

Kills the tmux session and removes the workspace from state.

## Keybindings (TUI)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate workspace list |
| `Enter` | Attach to selected workspace |
| `n` | New workspace |
| `d` | Delete workspace |
| `b` | Broadcast prompt |
| `D` | Diff viewer |
| `r` | Rename workspace |
| `/` | Filter workspaces |
| `Tab` | Cycle sidebar sections |
| `m` | Mark notifications read |
| `J` | Jump to next workspace needing attention |
| `R` | Restart agent |
| `s` | Stop agent |
| `?` | Help |
| `q` | Quit |

## Status Detection

Colosseum polls each workspace's agent pane via `tmux capture-pane` and matches the output against per-agent regex patterns:

| Status | Icon | Meaning |
|--------|------|---------|
| Working | `●` | Agent is actively processing (spinners, "esc to interrupt") |
| Waiting | `◉` | Agent needs input (permission prompts, questions) |
| Idle | `○` | Agent is at its prompt, ready for input |
| Stopped | `■` | tmux session no longer exists |
| Error | `✗` | Rate limits, auth failures, crashes |

Detection priority: Working > Waiting > Error > Idle.

## Supported Agents

| Agent | Binary | Auto-approve flag |
|-------|--------|-------------------|
| Claude Code | `claude` | `--dangerously-skip-permissions` |
| Codex CLI | `codex` | `--approval-mode full-auto` |

## Pane Layouts

- **`agent`** — Single pane running the agent
- **`agent-shell`** — Agent left, shell right (for dev servers, manual commands)
- **`agent-shell-logs`** — Agent left, shell + logs stacked right

## Architecture

```
cmd/colosseum/        CLI entry point (cobra)
internal/
  tmux/               tmux command abstraction (os/exec)
  agent/              Agent type definitions and detection patterns
  workspace/          Workspace model, persistence, lifecycle
  status/             Background status detection engine
  tui/                Bubble Tea TUI
    sidebar/          Workspace list panel
    preview/          Agent output preview panel
    theme/            Lipgloss styles
```

State is persisted to `~/.config/colosseum/workspaces.json`.

## Development

```bash
make build    # compile to ./colosseum
make test     # run all tests
make vet      # go vet ./...
make lint     # golangci-lint (if installed)
```

## License

MIT
