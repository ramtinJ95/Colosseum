# Colosseum Handoff Document

Plan-vs-implementation gap analysis. Covers every feature from `project-draft.md` against the current codebase as of 2026-03-04.

---

## What's Built

The project has a functional foundation consisting of 6 internal packages, a CLI entry point, and a Bubble Tea TUI that can create, list, delete, and switch between workspaces backed by tmux sessions.

| Metric | Value |
|--------|-------|
| Go source files (excluding worktree copies) | 32 |
| Test files | 8 |
| Test functions | 33 |
| Internal packages | 6 (`tmux`, `agent`, `workspace`, `status`, `tui`, `tui/theme`, `tui/preview`, `tui/sidebar`, `tui/dialog`) |
| CLI subcommands | 4 (`new`, `list`, `attach`, `delete`) |
| Fixture files | 18 `.txt` files across claude, codex, gemini |

**What the binary can do today:**

- Launch a Bubble Tea TUI dashboard inside tmux (`colosseum`)
- Create workspaces with a named agent, project path, branch, and pane layout via CLI (`colosseum new`) or TUI dialog (`n` key)
- Delete workspaces with tmux session cleanup via CLI (`colosseum delete`) or TUI dialog (`d` key)
- List workspaces with status icons via CLI (`colosseum list`)
- Attach to a workspace's tmux session via CLI (`colosseum attach`)
- Switch to a workspace from the TUI via Enter
- Poll agent pane output in the background and detect status (Working/Waiting/Idle/Error/Stopped) using per-agent regex patterns
- Display real-time status updates and pane preview content in the TUI
- Navigate the workspace list (j/k), jump to next workspace needing attention (J), and view a help overlay (?)

---

## Feature-by-Feature Status

### Workspace Management

| Feature | Status | Notes |
|---------|--------|-------|
| Create workspace with agent, path, branch | **Done** | `Manager.Create` builds tmux session, splits panes per layout, persists to JSON. |
| Delete workspace | **Done** | `Manager.Delete` kills tmux session and removes from store. |
| List workspaces | **Done** | `Store.List()` and CLI `list` command both implemented. |
| Switch to workspace | **Done** | `Manager.SwitchTo` calls `tmux switch-client`. TUI wires this to Enter key. |
| Rename workspace | **Not Started** | Key binding `r` is defined in `KeyMap` but no handler exists in `app.go`. No rename logic in `Manager` or `Store`. |
| Reorder workspaces | **Not Started** | No reordering logic exists anywhere. The store appends new workspaces and there is no move/swap operation. |
| Workspace persistence (JSON state file) | **Done** | `Store` reads/writes `~/.config/colosseum/workspaces.json` with atomic rename. Thread-safe with `sync.RWMutex`. |
| Configurable pane layouts | **Done** | Three layouts defined (`agent`, `agent-shell`, `agent-shell-logs`). `Manager.Create` splits panes accordingly. Layout selection is available in both CLI flags and TUI dialog. |

### Agent Support

| Feature | Status | Notes |
|---------|--------|-------|
| Pluggable agent registry | **Done** | Global registry in `agent/registry.go` with `Register`, `Get`, `Available`. |
| Per-agent binary, flags, YOLO flags | **Done** | `AgentDef` struct has `Binary`, `LaunchFlags`, `YoloFlags` for each agent. |
| 5 agents registered (Claude, Codex, Gemini, OpenCode, Aider) | **Done** | All five registered via `init()` functions. |
| Per-agent status detection patterns | **Done** | Each agent def has `WorkingPatterns`, `WaitingPatterns`, `IdlePatterns`, `ErrorPatterns`. |
| Custom instruction injection (--append-system-prompt) | **Not Started** | No custom instruction flags or logic. The Claude agent def has `--resume` in LaunchFlags but nothing for system prompt injection. |
| YOLO mode activation | **Partial** | YoloFlags are defined per agent (e.g., `--dangerously-skip-permissions` for Claude) but there is no mechanism to toggle or apply them. `Manager.Create` never launches the agent binary at all -- it only creates the tmux session and splits panes. |

### Status Detection

| Feature | Status | Notes |
|---------|--------|-------|
| Background goroutine polling | **Done** | `Poller.Run` ticks on a configurable interval, captures pane content, and sends `Update` structs through a channel. |
| Per-agent regex pattern matching | **Done** | `DetectFromContent` uses the agent def's patterns in priority order: Working > Waiting > Error > Idle. |
| Fixture-driven testing | **Done** | 18 fixture files across claude, codex, and gemini. `TestDetectFromContent_Fixtures` iterates all fixtures. |
| Working detection signal | **Done** | Patterns include `(esc to interrupt)`, braille spinner chars, and activity verbs. |
| Waiting detection signal | **Done** | Patterns include permission prompts, questions, y/n prompts. |
| Idle detection signal | **Done** | Patterns match prompt chars (`>`, `$`, etc.) at line start. |
| Error detection signal | **Done** | Shared patterns for rate limits, panics, auth errors. |
| Stopped detection signal | **Done** | When `CapturePane` returns an error (session gone), detector returns `StatusStopped`. |
| OpenCode/Aider fixture tests | **Not Started** | No fixture files exist under `testdata/fixtures/opencode/` or `testdata/fixtures/aider/`. Detector test only covers Claude, Codex, and Gemini. |

### Git Worktree Integration

| Feature | Status | Notes |
|---------|--------|-------|
| Create git worktree per workspace | **Not Started** | The draft lists an `internal/worktree/` package. No such package exists. The workspace model stores a `Branch` field but nothing calls `git worktree add`. |
| Configurable worktree path templates | **Not Started** | No template resolution logic. |
| Broadcast worktrees (one branch per workspace) | **Not Started** | No broadcast-related worktree creation. |
| Cleanup worktree on deletion | **Not Started** | `Manager.Delete` kills the tmux session but does not touch git worktrees. |
| Diff viewer (worktree branch vs base) | **Not Started** | No diff computation code. The draft lists `internal/worktree/diff.go` -- this file does not exist. |

### Notification System

| Feature | Status | Notes |
|---------|--------|-------|
| In-memory notification store | **Not Started** | The draft lists `internal/notification/` package. No such package exists. |
| Per-workspace unread counts | **Partial** | The `Workspace` struct has an `UnreadCount` field and the sidebar renders unread badges, but nothing increments `UnreadCount`. It is always 0. |
| Status transition notifications | **Not Started** | The poller sends `Update` events with previous/current status but no code translates these into notification objects. |
| Desktop notifications (notify-send) | **Not Started** | No `notify-send` integration. |
| Claude Code hook integration | **Not Started** | The draft lists `internal/hook/` package. No such package exists. |
| Jump-to-unread (J key) | **Partial** | The `J` key handler in `app.go` jumps to the next workspace with `StatusWaiting` or `StatusError`. However, it navigates by status, not by unread notifications. Close to the draft intent but not notification-aware. |
| Mark notifications as read (m key) | **Not Started** | Key binding defined in `KeyMap` but no handler in `app.go`. |
| Notification section in sidebar | **Not Started** | Sidebar only renders the workspace list. No notification section. |
| Tab to cycle sidebar sections | **Not Started** | Key binding defined but no handler. The sidebar has no section concept. |

### Broadcast Prompt

| Feature | Status | Notes |
|---------|--------|-------|
| Broadcast prompt dialog | **Not Started** | Key binding `b` is defined in `KeyMap` but no handler in `app.go`. The draft lists `dialog/broadcast.go` -- this file does not exist. |
| Multi-select workspace picker | **Not Started** | No multi-select component. |
| Auto-create worktrees on broadcast | **Not Started** | Requires worktree integration which is also not built. |
| Send prompt via tmux send-keys to multiple panes | **Not Started** | `Client.SendKeys` exists and works, but no broadcast orchestration calls it for multiple workspaces. |
| CLI `broadcast` command | **Not Started** | Not registered in `main.go`. |

### Diff Viewer

| Feature | Status | Notes |
|---------|--------|-------|
| Full-screen diff overlay | **Not Started** | Key binding `D` is defined but no handler in `app.go`. The draft lists `tui/diff/model.go` and `tui/diff/view.go` -- neither exists. |
| Side-by-side diff rendering | **Not Started** | No diff rendering code. |
| File list panel with navigation | **Not Started** | No diff file list. |
| Hunk scroll with configurable context | **Not Started** | No hunk navigation. |
| CLI `diff` command | **Not Started** | Not registered in `main.go`. |

### TUI Dashboard

| Feature | Status | Notes |
|---------|--------|-------|
| Bubble Tea app with sidebar + preview | **Done** | `tui/app.go` composes `sidebar.Model` and `preview.Model` side by side using lipgloss. |
| Sidebar workspace list with status icons | **Done** | Sidebar renders status icons, agent type, branch name, and unread badges per workspace. |
| Preview panel with captured pane output | **Done** | Preview panel shows the last N lines of the selected workspace's agent pane, refreshed on poll updates and cursor navigation. |
| New workspace dialog (name, path, agent, branch, layout) | **Done** | Full form with text inputs and selector widgets for agent type and layout. |
| Delete confirmation dialog | **Done** | Prompts for y/n confirmation before killing session. |
| Help overlay | **Done** | Lists keybindings. Currently only shows 7 of the planned 15+ keybindings. |
| Lipgloss styling with theme | **Done** | `theme.Theme` struct with full style definitions for all status types, borders, titles, badges, help text. |
| Theme support (catppuccin, switching) | **Not Started** | Only `DefaultTheme()` exists. No theme loading from config, no switching. |

### CLI Commands

| Command | Status | Notes |
|---------|--------|-------|
| `colosseum` (launch TUI) | **Done** | Root command wires up store, tmux client, manager, poller, and launches Bubble Tea with alt screen. |
| `colosseum new` | **Done** | Accepts `--path`, `--agent`, `--branch`, `--layout` flags. Validates agent type. |
| `colosseum list` | **Done** | Lists workspaces with status icons, branch, agent type. |
| `colosseum attach` | **Done** | Finds workspace by name and calls `SwitchTo`. |
| `colosseum delete` | **Done** | Finds workspace by name and calls `Delete`. |
| `colosseum broadcast` | **Not Started** | Not registered. |
| `colosseum diff` | **Not Started** | Not registered. |

### Keybindings

| Key | Action | Status | Notes |
|-----|--------|--------|-------|
| `j/k` | Navigate workspace list | **Done** | Sidebar model handles j/k for cursor movement. |
| `Enter` | Attach to selected workspace | **Done** | Calls `switchToWorkspace` which runs `tmux switch-client`. |
| `n` | New workspace dialog | **Done** | Opens `NewWorkspaceModel` dialog. |
| `d` | Delete workspace | **Done** | Opens `DeleteModel` confirmation dialog. |
| `b` | Broadcast prompt dialog | **Not Started** | Binding defined, no handler. |
| `D` | Diff viewer | **Not Started** | Binding defined, no handler. |
| `r` | Rename workspace | **Not Started** | Binding defined, no handler. |
| `/` | Filter/search workspaces | **Not Started** | Binding defined, no handler. |
| `Tab` | Cycle sidebar sections | **Not Started** | Binding defined, no handler. |
| `m` | Mark notifications as read | **Not Started** | Binding defined, no handler. |
| `J` | Jump to next needing attention | **Done** | Implemented in `jumpToNextAttention()`. Scans for Waiting/Error status. |
| `R` | Restart agent | **Not Started** | Binding defined, no handler. |
| `s` | Stop agent | **Not Started** | Binding defined, no handler. |
| `?` | Help overlay | **Done** | Opens `HelpModel` dialog. |
| `q` | Quit | **Done** | Sends `tea.Quit`. |
| `up/down` arrow keys | Navigate | **Not Started** | Only `j/k` is wired in the sidebar. Arrow keys are not in the sidebar model's Update handler. |

### Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| TOML config file | **Not Started** | The draft lists `internal/config/` package with `config.go` and `paths.go`. No such package exists. `go-toml` is listed as a dependency in the draft but is not in `go.mod`. |
| Config-driven poll interval | **Not Started** | Poll interval is hardcoded to `1500ms` in `main.go`. |
| Config-driven default agent/layout | **Not Started** | Defaults are hardcoded in CLI flags. |
| Config-driven worktree settings | **Not Started** | No worktree config. |
| Config-driven notification settings | **Not Started** | No notification config. |
| Config-driven theme name | **Not Started** | No theme config. |

---

## Known Issues

1. **Unhandled keybindings**: 8 key bindings are defined in `KeyMap` (`Broadcast`, `Diff`, `Rename`, `Filter`, `Tab`, `MarkRead`, `Restart`, `Stop`) but have no corresponding `case` in `app.go`'s `updateNormal`. Pressing these keys does nothing silently.

2. **Arrow key navigation missing**: The sidebar's `Update` method only handles `j` and `k` strings. The draft specifies arrow keys should also work.

3. **Agent binary never launched**: `Manager.Create` creates a tmux session and splits panes but never launches the agent binary (e.g., `claude`, `codex`). The agent pane starts with an empty shell. The user must manually start the agent.

4. **UnreadCount always zero**: The `Workspace.UnreadCount` field exists and the sidebar renders it, but nothing ever writes to it. The poller emits status transitions but no code increments unread counts.

5. **Help overlay incomplete**: The help dialog only lists 7 keybindings. The KeyMap defines 16. Users cannot discover the full set of available actions.

6. **No error handling on poller list failure**: `Poller.poll` silently returns on `provider.List()` error with no logging or status update.

7. **StatusUpdateMsg listener re-registration**: `listenForUpdates` reads one update from the channel, then must be re-registered via `tea.Cmd`. If the TUI is in a dialog state (new workspace, delete, help), status updates are received via the top-level `Update` switch but then the method re-registers. This works but status updates during dialog interaction cause the dialog to re-render due to the `tea.Batch` returning commands.

8. **No duplicate workspace name check**: `Manager.Create` does not check if a workspace with the same title already exists. Two workspaces can share the same session name, causing a tmux collision.

9. **Missing `sergi/go-diff` dependency**: The draft lists `github.com/sergi/go-diff` for diff computation. It is not in `go.mod` and no diff code exists.

10. **Missing `go-toml` dependency**: The draft lists `github.com/pelletier/go-toml/v2` for config parsing. It is not in `go.mod` and no config code exists.

11. **Delete does not handle tmux session already killed**: If the tmux session was already killed externally, `Manager.Delete` will return an error from `KillSession` and leave the workspace in the JSON store.

---

## Recommended Next Steps

Ordered by impact and dependency chain (earlier items unblock later items):

1. **Launch agent binary on workspace creation** -- Without this, every workspace requires manual agent startup. Add a `SendKeys` call after session creation to run the agent binary with appropriate flags. This is the single highest-impact gap.

2. **Add `internal/config/` package with TOML loading** -- Many features depend on configuration (poll interval, default agent, worktree paths, notification toggles, theme). Wiring config early avoids hardcoded values spreading further.

3. **Implement `internal/worktree/` package** -- Git worktree create/remove/list. This unblocks broadcast and diff features, which are the project's core differentiators per the vision statement.

4. **Implement `internal/notification/` package** -- In-memory store, status-transition-to-notification logic, unread count management. Wire the poller's `Update` events to notification creation.

5. **Wire remaining keybindings** -- Low effort, high discoverability improvement:
   - `r` (rename): Add `Rename` method to `Manager`, create a rename dialog.
   - `s` (stop): Send Ctrl+C to agent pane via `tmux send-keys`.
   - `R` (restart): Kill and re-send the agent launch command.
   - `m` (mark read): Clear unread count for selected workspace.
   - `Tab` (cycle sections): Add notification section to sidebar.
   - `/` (filter): Add text input filter over workspace list.
   - Arrow keys: Add `up`/`down` to sidebar model.

6. **Broadcast prompt dialog and execution** -- Create `dialog/broadcast.go` with multi-select workspace picker, wire `b` key. Use `SendKeys` to dispatch the prompt to each selected workspace's agent pane.

7. **Diff viewer** -- Create `tui/diff/` package, add `internal/worktree/diff.go` for git diff computation, wire `D` key. Requires worktree package from step 3.

8. **Desktop notifications** -- Implement `notify-send` integration. Depends on notification store from step 4.

9. **OpenCode and Aider fixture tests** -- Add fixture files for the remaining two agents. The detector test infrastructure already supports it; only the fixture data is missing.

10. **Duplicate workspace name guard** -- Add a uniqueness check in `Manager.Create` before creating the tmux session.

11. **Theme support and config-driven theming** -- Add alternative themes (catppuccin), load theme name from config.
