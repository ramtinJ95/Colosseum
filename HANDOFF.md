# Colosseum Handoff Document

Plan-vs-implementation gap analysis. Covers every feature from `project-draft.md` against the current codebase as of 2026-03-04.

---

## What's Built

The project has a functional foundation consisting of 6 internal packages, a CLI entry point, and a Bubble Tea TUI that can create, list, delete, and switch between workspaces backed by tmux sessions. Agents auto-launch on workspace creation and status is detected in real time.

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
- Auto-launch the agent binary (e.g., `claude`, `codex`, `gemini`) in the agent pane on workspace creation
- Delete workspaces with tmux session cleanup via CLI (`colosseum delete`) or TUI dialog (`d` key) — gracefully handles already-killed sessions
- List workspaces with status icons via CLI (`colosseum list`)
- Attach to a workspace's tmux session via CLI (`colosseum attach`) or TUI (`Enter`)
- Poll agent pane output in the background and detect status (Working/Waiting/Idle/Error/Stopped) using per-agent regex patterns
- Display real-time status updates and pane preview content in the TUI
- Switch preview between panes (agent/shell/logs) using `h`/`l` keys with a visual tab bar
- Navigate the workspace list (`j`/`k`), jump to next workspace needing attention (`J`), and view a help overlay (`?`)
- Path autocomplete in the new workspace dialog (filesystem directory suggestions via Tab)
- Duplicate workspace name guard prevents tmux session collisions

---

## Feature-by-Feature Status

### Workspace Management

| Feature | Status | Notes |
|---------|--------|-------|
| Create workspace with agent, path, branch | **Done** | `Manager.Create` builds tmux session, splits panes per layout, launches agent, persists to JSON. |
| Auto-launch agent on create | **Done** | Agent binary is run via `tmux send-keys` in the agent pane immediately after session setup. Uses `AgentDef.Binary` + `LaunchFlags`. |
| Delete workspace | **Done** | `Manager.Delete` kills tmux session (ignoring errors if already dead) and removes from store. |
| Duplicate name guard | **Done** | `Manager.Create` checks for existing workspace with same title before creating. |
| List workspaces | **Done** | `Store.List()` and CLI `list` command both implemented. |
| Switch to workspace | **Done** | `Manager.SwitchTo` calls `tmux switch-client`. TUI wires this to Enter key. Status bar shows `tmux prefix+L` hint for returning. |
| Rename workspace | **Not Started** | Key binding `r` is defined in `KeyMap` but no handler exists in `app.go`. No rename logic in `Manager` or `Store`. |
| Reorder workspaces | **Not Started** | No reordering logic exists anywhere. The store appends new workspaces and there is no move/swap operation. |
| Workspace persistence (JSON state file) | **Done** | `Store` reads/writes `~/.config/colosseum/workspaces.json` with atomic rename. Thread-safe with `sync.RWMutex`. |
| Configurable pane layouts | **Done** | Three layouts defined (`agent`, `agent-shell`, `agent-shell-logs`). `Manager.Create` splits panes accordingly. Layout selection is available in both CLI flags and TUI dialog. |

### Agent Support

| Feature | Status | Notes |
|---------|--------|-------|
| Pluggable agent registry | **Done** | Global registry in `agent/registry.go` with `Register`, `Get`, `Available`. |
| Per-agent binary and flags | **Done** | `AgentDef` struct has `Binary`, `LaunchFlags`, `YoloFlags` for each agent. All agents launch with just their binary name (no default flags). |
| 5 agents registered | **Done** | Claude (`claude`), Codex (`codex`), Gemini (`gemini`), OpenCode (`opencode`), Aider (`aider`). All registered via `init()` functions. |
| Per-agent status detection patterns | **Done** | Each agent def has `WorkingPatterns`, `WaitingPatterns`, `IdlePatterns`, `ErrorPatterns`. |
| YOLO mode activation | **Partial** | YoloFlags are defined per agent (e.g., `--dangerously-skip-permissions` for Claude, `--approval-mode full-auto` for Codex) but there is no UI toggle or mechanism to apply them. The new workspace dialog does not expose a YOLO option. |
| Custom instruction injection | **Not Started** | No custom instruction flags or logic. |

### Status Detection

| Feature | Status | Notes |
|---------|--------|-------|
| Background goroutine polling | **Done** | `Poller.Run` ticks on a configurable interval, captures pane content, and sends `Update` structs through a channel. |
| Per-agent regex pattern matching | **Done** | `DetectFromContent` uses the agent def's patterns in priority order: Working > Waiting > Error > Idle. |
| Fixture-driven testing | **Done** | 18 fixture files across claude, codex, and gemini. `TestDetectFromContent_Fixtures` iterates all fixtures. |
| Pane targeting | **Done** | Uses real tmux pane IDs (`%N` format) returned from `new-session -P -F #{pane_id}`. Works regardless of tmux `base-index` setting. |
| Poller error handling | **Done** | On `List()` failure, all tracked workspaces transition to `StatusStopped` with updates sent to the channel. |
| OpenCode/Aider fixture tests | **Not Started** | No fixture files exist under `testdata/fixtures/opencode/` or `testdata/fixtures/aider/`. Detector test only covers Claude, Codex, and Gemini. |

### Git Worktree Integration

| Feature | Status | Notes |
|---------|--------|-------|
| Create git worktree per workspace | **Not Started** | The draft lists an `internal/worktree/` package. No such package exists. The workspace model stores a `Branch` field but nothing calls `git worktree add`. |
| Configurable worktree path templates | **Not Started** | No template resolution logic. |
| Broadcast worktrees | **Not Started** | No broadcast-related worktree creation. |
| Cleanup worktree on deletion | **Not Started** | `Manager.Delete` kills the tmux session but does not touch git worktrees. |
| Diff viewer (worktree branch vs base) | **Not Started** | No diff computation code. |

### Notification System

| Feature | Status | Notes |
|---------|--------|-------|
| In-memory notification store | **Not Started** | No `internal/notification/` package exists. |
| Per-workspace unread counts | **Partial** | The `Workspace` struct has an `UnreadCount` field and the sidebar renders unread badges, but nothing increments `UnreadCount`. It is always 0. |
| Status transition notifications | **Not Started** | The poller sends `Update` events with previous/current status but no code translates these into notification objects. |
| Desktop notifications (notify-send) | **Not Started** | No `notify-send` integration. |
| Claude Code hook integration | **Not Started** | No `internal/hook/` package exists. |
| Jump-to-unread (J key) | **Partial** | Jumps to the next workspace with `StatusWaiting` or `StatusError`. Navigates by status, not by unread notifications. |
| Mark notifications as read (m key) | **Not Started** | Key binding defined, no handler. |

### Broadcast Prompt

| Feature | Status | Notes |
|---------|--------|-------|
| Broadcast prompt dialog | **Not Started** | Key binding `b` is defined but no handler. No `dialog/broadcast.go` exists. |
| Multi-select workspace picker | **Not Started** | No multi-select component. |
| Send prompt to multiple panes | **Not Started** | `Client.SendKeys` exists and works, but no broadcast orchestration. |
| CLI `broadcast` command | **Not Started** | Not registered in `main.go`. |

### Diff Viewer

| Feature | Status | Notes |
|---------|--------|-------|
| Full-screen diff overlay | **Not Started** | Key binding `D` is defined but no handler. No `tui/diff/` package exists. |
| Side-by-side diff rendering | **Not Started** | No diff rendering code. |
| CLI `diff` command | **Not Started** | Not registered in `main.go`. |

### TUI Dashboard

| Feature | Status | Notes |
|---------|--------|-------|
| Bubble Tea app with sidebar + preview | **Done** | `tui/app.go` composes `sidebar.Model` and `preview.Model` side by side using lipgloss. |
| Sidebar workspace list with status icons | **Done** | Sidebar renders status icons, agent type, branch name, and unread badges per workspace. |
| Preview panel with pane output | **Done** | Preview panel shows the last 50 lines of the selected pane, refreshed on poll updates and cursor navigation. |
| Pane tab bar in preview | **Done** | Multi-pane layouts show a tab bar (`[agent] shell logs`) with h/l switching. Active tab highlighted. Hidden for single-pane layouts. |
| New workspace dialog | **Done** | Full form with text inputs (name, path with autocomplete, branch) and selectors (agent, layout). Tab/Enter navigation. |
| Path autocomplete | **Done** | Path field uses `textinput.ShowSuggestions` with dynamic filesystem directory suggestions. Tab accepts suggestion, `~/` expansion supported. Hidden dirs excluded unless typing a dot. |
| Delete confirmation dialog | **Done** | Prompts for y/n confirmation before killing session. |
| Help overlay | **Done** | Lists all keybindings including unimplemented ones, plus `tmux prefix+L` return hint. |
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
| `h/l` | Switch preview pane tab | **Done** | Cycles through available panes (agent/shell/logs) with visual tab bar. |
| `Enter` | Attach to selected workspace | **Done** | Calls `switchToWorkspace` which runs `tmux switch-client`. Shows return hint. |
| `n` | New workspace dialog | **Done** | Opens `NewWorkspaceModel` dialog with path autocomplete. |
| `d` | Delete workspace | **Done** | Opens `DeleteModel` confirmation dialog. |
| `J` | Jump to next needing attention | **Done** | Scans for Waiting/Error status. Resets pane focus to agent. |
| `?` | Help overlay | **Done** | Opens `HelpModel` dialog with all bindings listed. |
| `q` | Quit | **Done** | Sends `tea.Quit`. |
| `b` | Broadcast prompt dialog | **Not Started** | Binding defined, no handler. |
| `D` | Diff viewer | **Not Started** | Binding defined, no handler. |
| `r` | Rename workspace | **Not Started** | Binding defined, no handler. |
| `/` | Filter/search workspaces | **Not Started** | Binding defined, no handler. |
| `m` | Mark notifications as read | **Not Started** | Binding defined, no handler. |
| `R` | Restart agent | **Not Started** | Binding defined, no handler. |
| `s` | Stop agent | **Not Started** | Binding defined, no handler. |

### Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| TOML config file | **Not Started** | No `internal/config/` package exists. `go-toml` is not in `go.mod`. |
| Config-driven poll interval | **Not Started** | Poll interval is hardcoded to `1500ms` in `main.go`. |
| Config-driven default agent/layout | **Not Started** | Defaults are hardcoded in CLI flags. |
| Config-driven worktree settings | **Not Started** | No worktree config. |
| Config-driven notification settings | **Not Started** | No notification config. |
| Config-driven theme name | **Not Started** | No theme config. |

---

## Known Issues

1. **Unhandled keybindings**: 7 key bindings are defined in `KeyMap` (`Broadcast`, `Diff`, `Rename`, `Filter`, `MarkRead`, `Restart`, `Stop`) but have no corresponding handler in `app.go`. Pressing these keys does nothing silently.

2. **UnreadCount always zero**: The `Workspace.UnreadCount` field exists and the sidebar renders it, but nothing ever writes to it. The poller emits status transitions but no code increments unread counts.

3. **YoloFlags never applied**: YoloFlags are defined per agent but there is no UI toggle or code path to apply them when launching an agent.

4. **StatusUpdateMsg listener re-registration**: `listenForUpdates` reads one update from the channel, then must be re-registered via `tea.Cmd`. This works correctly but status updates during dialog interaction cause a re-render via `tea.Batch`.

5. **Missing `sergi/go-diff` dependency**: The draft lists `github.com/sergi/go-diff` for diff computation. It is not in `go.mod` and no diff code exists.

6. **Missing `go-toml` dependency**: The draft lists `github.com/pelletier/go-toml/v2` for config parsing. It is not in `go.mod` and no config code exists.

7. **SendKeys ignores errors on agent launch**: `Manager.Create` discards the error from `SendKeys` when launching the agent binary. If the agent binary is not installed, the error is silent.

---

## Resolved Issues (from prior handoff)

These issues from the previous handoff have been fixed:

1. ~~Agent binary never launched~~ — Manager.Create now launches the agent via SendKeys after session setup.
2. ~~Pane target hardcoded as session:0.0~~ — CreateSession returns real pane IDs via `-P -F #{pane_id}`.
3. ~~Dialog navigation hijacks text input~~ — j/k/h/l removed from dialog field navigation; tab/shift+tab/enter used instead.
4. ~~Delete fails on already-killed sessions~~ — KillSession error is now ignored; workspace always removed from store.
5. ~~No duplicate workspace name check~~ — Manager.Create checks for existing title before creating.
6. ~~Help overlay incomplete~~ — All keybindings now listed including unimplemented ones.
7. ~~No error handling on poller list failure~~ — Poller transitions all tracked workspaces to Stopped on error.
8. ~~Arrow key navigation~~ — Intentionally vim-only (j/k/h/l) per user preference.

---

## Recommended Next Steps

Ordered by impact and dependency chain:

1. **Add `internal/config/` package with TOML loading** — Many features depend on configuration (poll interval, default agent, worktree paths, notification toggles, theme). Wiring config early avoids hardcoded values spreading further.

2. **Implement `internal/worktree/` package** — Git worktree create/remove/list. This unblocks broadcast and diff features, which are the project's core differentiators per the vision statement.

3. **Implement `internal/notification/` package** — In-memory store, status-transition-to-notification logic, unread count management. Wire the poller's `Update` events to notification creation.

4. **Wire remaining keybindings** — Low effort, high discoverability:
   - `s` (stop): Send Ctrl+C to agent pane via `tmux send-keys`.
   - `R` (restart): Kill and re-send the agent launch command.
   - `r` (rename): Add `Rename` method to `Manager`, create a rename dialog.
   - `m` (mark read): Clear unread count for selected workspace.
   - `/` (filter): Add text input filter over workspace list.

5. **YOLO mode toggle** — Add a checkbox or toggle in the new workspace dialog and/or a keybinding to enable YOLO mode (appends `YoloFlags` to the agent launch command).

6. **Broadcast prompt dialog and execution** — Create `dialog/broadcast.go` with multi-select workspace picker, wire `b` key. Use `SendKeys` to dispatch the prompt to each selected workspace's agent pane.

7. **Diff viewer** — Create `tui/diff/` package, wire `D` key. Requires worktree package from step 2.

8. **Desktop notifications** — Implement `notify-send` integration. Depends on notification store from step 3.

9. **OpenCode and Aider fixture tests** — Add fixture files for the remaining two agents. The detector test infrastructure already supports it; only the fixture data is missing.

10. **Theme support and config-driven theming** — Add alternative themes (catppuccin), load theme name from config.
