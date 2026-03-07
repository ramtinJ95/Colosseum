# Colosseum Handoff Document

Current implementation snapshot and gap analysis for the codebase as of 2026-03-07.

This document is meant to be a trustworthy handoff for the next agent. It reflects the current repo state after the recent hardening work, not just the original `project-draft.md` plan.

---

## Current Snapshot

Colosseum is a Go + tmux + Bubble Tea workspace manager for running AI coding agents in parallel. The current product shape is:

- A working tmux-backed dashboard and CLI for creating, listing, attaching to, and deleting workspaces
- Real status detection via pane scraping with per-agent regex patterns tuned to current Claude and Codex terminal output
- A supported new-workspace surface limited to `claude` and `codex`
- A shell-style path-completion flow in the new-workspace dialog
- A deterministic tmux return path from attached workspaces back to the dashboard session that launched Colosseum
- A still-unimplemented roadmap for worktrees, broadcast, notifications, diffing, and config

### Repo Metrics

| Metric | Value |
|--------|-------|
| Go source files under `cmd/` + `internal/` | 44 |
| Test files | 12 |
| Test functions | 55 |
| Go packages under `cmd/` + `internal/` | 10 |
| CLI subcommands | 4 (`new`, `list`, `attach`, `delete`) |
| Fixture files | 24 `.txt` files |

### Recent Changes Since The Previous Handoff

**Status detection hardening (2026-03-07)** — researched cmux, agent-of-empires, hive, agent-deck, and claude-squad to learn how other tmux-based agent managers detect agent state. Applied four improvements:

13. **ANSI stripping**: New `StripANSI()` in `internal/status/normalize.go` strips CSI, OSC, and simple escape sequences from pane content before pattern matching. Single-pass O(n) parser, no regex.
14. **Detection window 10→30 lines**: `lastNonEmptyLines` increased from 10 to 30 non-empty lines so questions asked further back aren't lost from the analysis window.
15. **Pane title detection**: `PaneCapturer` interface extended with `CapturePaneTitle()`. Claude Code sets braille spinner chars (U+2800-U+28FF) in the tmux pane title while working. Used as a supplementary Working signal that upgrades `StatusUnknown` only — never overrides Idle/Waiting/Error (title is sticky and may not be cleared after a crash).
16. **Spike/hysteresis filtering**: Poller now requires non-urgent states to persist for 1s (spike window) before confirming a transition, with a 500ms minimum hold on the current state (hysteresis). Urgent statuses (Waiting, Error, Stopped) and initial detection bypass both filters. Configurable via `WithSpikeWindow()`/`WithHysteresisWindow()` options.
17. **Tiered window narrowing**: In the non-idle detection path, Waiting patterns check only the last 10 lines (not all 30). Working and Error still check all 30. This matches hive (15 lines) and agent-deck (tiered 15/10/5/3/1).
18. **Tighter waiting patterns**: Removed broad `\?\s*$` from Claude and Codex `WaitingPatterns` (no other project uses such a vague pattern). The question-mark check is preserved only in the idle path's 3-line window for catching natural-language questions at the prompt.
19. **Race fix**: `mockCapturer` in poller tests now uses `sync.Mutex` and `SetContent()` for safe concurrent access — confirmed clean with `go test -race`.

These changes landed before the 2026-03-04 items below:

1. New workspace creation is now intentionally restricted to `claude` and `codex`.
2. The new-workspace path field now behaves more like a shell:
   - live directory suggestions still update while typing
   - `Enter` advances/submits instead of implicitly accepting completion
   - `Tab` expands to the longest shared path prefix, then cycles matches on repeated presses
   - `~` / `~/...` inputs are expanded before workspace creation
   - the current suggestion is still rendered explicitly in the dialog
3. Workspace lifecycle is safer:
   - invalid layouts fail fast
   - unsupported agents fail fast
   - failed workspace creation rolls back the tmux session
   - agent launch errors are surfaced instead of ignored
   - delete only ignores real “session not found” errors
4. `colosseum list` now refreshes live status from tmux before printing.
5. Initial TUI workspace load now refreshes live status from tmux instead of trusting stale JSON.
6. `tmux send-keys` for command dispatch now sends literal text first and `Enter` separately.
7. Unimplemented roadmap keybindings no longer silently no-op:
   - pressing them shows a status-bar message
   - the help overlay marks them unavailable
8. The preview pane now refreshes on a timer instead of only on status transitions or cursor movement.
9. Preview content now wraps to the viewport width instead of overflowing past the pane border.
10. Attaching to a workspace now installs a deterministic tmux return binding:
    - `prefix+e` switches back to the dashboard session that launched Colosseum
    - this is not tied to a hardcoded `colo-ctrl` session name
11. Status detection was hardened against current Claude and Codex CLI drift:
    - Codex now recognizes current `Working (... esc to interrupt)` output and current footer lines
    - Claude now recognizes current `✻ Cooked for ...` activity lines and current footer/status-bar lines
    - a bottom prompt can now resolve to `Working`, `Waiting`, or `Idle` based on immediately recent lines
    - prompt-plus-recent-question states are now treated as `Waiting` consistently across Claude and Codex
12. Fixture coverage and tests were expanded to cover the current live Claude/Codex pane formats plus preview wrapping.

---

## What Works Today

The binary can currently:

- Launch the Bubble Tea dashboard inside tmux via `colosseum`
- Create a workspace from CLI or TUI
- Create pane layouts `agent`, `agent-shell`, or `agent-shell-logs`
- Auto-launch the selected agent in the agent pane
- Persist workspace state to `~/.config/colosseum/workspaces.json`
- Refresh and display live workspace status
- Attach to an existing workspace tmux session
- Return from an attached workspace to the dashboard session with `prefix+e`
- Delete a workspace and clean up the tmux session
- Preview agent, shell, or logs panes inside the dashboard
- Offer shell-style path completion in the new-workspace dialog

The current supported agent surface for new workspaces is:

| Agent | Status |
|-------|--------|
| Claude Code | Supported for creation |
| Codex CLI | Supported for creation |
| Gemini | Legacy definition still registered, not supported for new workspace creation |
| OpenCode | Legacy definition still registered, not supported for new workspace creation |
| Aider | Legacy definition still registered, not supported for new workspace creation |

The “legacy definition still registered” detail matters because existing saved workspaces using those agent types can still be read and status-detected, but the normal create flow will reject them.

---

## Feature-By-Feature Status

### Workspace Management

| Feature | Status | Notes |
|---------|--------|-------|
| Create workspace with agent, path, branch | **Done** | `Manager.Create` now validates agent support and layout before creating anything. |
| Transactional create behavior | **Done** | If pane creation, launch, or persistence fails, the tmux session is rolled back. |
| Auto-launch agent on create | **Done** | Agent launch errors are now returned instead of silently ignored. |
| Delete workspace | **Done** | Deletes the JSON record only if tmux cleanup succeeds or the session is already gone. |
| Duplicate name guard | **Done** | Title collision is checked before creation. |
| List workspaces | **Done** | CLI list now refreshes live status from tmux before printing. |
| Switch to workspace | **Done** | `tmux switch-client` is wired through CLI and TUI. |
| Rename workspace | **Not Started** | No rename logic or dialog exists yet. |
| Reorder workspaces | **Not Started** | No move/swap or ordering metadata exists. |
| Workspace persistence | **Done** | JSON persistence with atomic rename and mutex protection. |
| Configurable pane layouts | **Done** | Three layouts supported; invalid CLI layout strings are rejected. |

### Agent Support

| Feature | Status | Notes |
|---------|--------|-------|
| Pluggable agent registry | **Done** | `Register`, `Get`, and `Available` still exist. |
| Supported-agent surface | **Done** | `Supported()` / `IsSupported()` limit creation to `claude` and `codex`. |
| Legacy registry entries for 5 agents | **Done** | All five definitions are still registered for compatibility and detection. |
| Per-agent binary and flags | **Done** | Definitions include binary, launch flags, yolo flags, and detection patterns. |
| Per-agent status detection patterns | **Done** | Claude, Codex, Gemini, OpenCode, and Aider all have pattern definitions. |
| YOLO mode activation | **Partial** | Flags exist on agent defs, but there is still no UI or CLI switch to apply them. |
| Custom instruction injection | **Not Started** | No implementation yet. |

### Status Detection

| Feature | Status | Notes |
|---------|--------|-------|
| Background polling | **Done** | `Poller.Run` emits status updates on an interval. |
| ANSI stripping | **Done** | `StripANSI()` in `normalize.go` strips escape sequences before pattern matching. |
| Per-agent regex detection | **Done** | Bottom prompt lines now resolve contextually: visible recent work keeps `Working`, visible recent question/choice state yields `Waiting`, otherwise prompt-only falls back to `Idle`. |
| Tiered window narrowing | **Done** | 30-line window for Working/Error, 10-line window for Waiting in non-idle path, 3-line window for idle-path question detection. |
| Pane title detection | **Done** | Braille in tmux pane title upgrades `Unknown` → `Working`. Does not override `Idle` (title is sticky). |
| Spike/hysteresis filtering | **Done** | 1s spike + 500ms hysteresis prevents flicker. Urgent states (Waiting/Error/Stopped) bypass. Configurable via `WithSpikeWindow`/`WithHysteresisWindow`. |
| Current Claude/Codex CLI drift coverage | **Done** | Detection rules now cover current Claude `✻ Cooked for ...` lines, current Claude footer chrome, current Codex `Working (... esc to interrupt)` lines, and current Codex footer chrome. |
| Fixture-driven testing | **Done** | Fixture coverage exists for Claude, Codex, and Gemini, including current live Claude/Codex pane formats. |
| Pane targeting | **Done** | Uses real pane IDs returned by tmux. |
| Poller error handling | **Done** | Provider failure transitions tracked workspaces to `Stopped`. |
| Poller stale-status cleanup | **Done** | Status entries for deleted workspaces are now removed from the in-memory poller map. |
| CLI live status refresh | **Done** | `colosseum list` does not rely on stale persisted status anymore. |
| Initial TUI live status refresh | **Done** | Initial sidebar load refreshes statuses before rendering. |
| OpenCode/Aider fixture tests | **Not Started** | Definitions exist, but there are still no dedicated fixtures for them. |

### Git Worktree Integration

| Feature | Status | Notes |
|---------|--------|-------|
| Create git worktree per workspace | **Not Started** | No `internal/worktree/` package exists. |
| Configurable worktree path templates | **Not Started** | No template resolution logic. |
| Cleanup worktree on deletion | **Not Started** | Delete only handles tmux and JSON state. |
| Broadcast worktrees | **Not Started** | No orchestration exists. |
| Diff viewer against base branch | **Not Started** | No diff computation package exists. |

### Notification System

| Feature | Status | Notes |
|---------|--------|-------|
| Notification store | **Not Started** | No `internal/notification/` package exists. |
| Per-workspace unread counts | **Partial** | `UnreadCount` exists on the model and renders in the sidebar, but no code increments it. |
| Status transition notifications | **Not Started** | Poller emits updates, but nothing converts them into notification records. |
| Desktop notifications | **Not Started** | No `notify-send` integration. |
| Claude hook integration | **Not Started** | No `internal/hook/` package exists. |

### Broadcast Prompt

| Feature | Status | Notes |
|---------|--------|-------|
| Broadcast dialog | **Not Started** | No dialog, model, or dispatch logic exists. |
| Multi-select target picker | **Not Started** | No component exists. |
| Prompt fan-out via tmux | **Not Started** | `SendKeys` exists, but no broadcast orchestration uses it. |
| CLI `broadcast` command | **Not Started** | Not registered. |

### Diff Viewer

| Feature | Status | Notes |
|---------|--------|-------|
| Full-screen diff overlay | **Not Started** | No `tui/diff/` package exists. |
| Side-by-side diff rendering | **Not Started** | Not implemented. |
| CLI `diff` command | **Not Started** | Not registered. |

### TUI Dashboard

| Feature | Status | Notes |
|---------|--------|-------|
| Sidebar + preview layout | **Done** | Main app composes sidebar and preview with lipgloss. |
| Sidebar workspace list with status icons | **Done** | Includes branch and unread badge rendering. |
| Preview panel | **Done** | Shows selected pane content, wraps long lines to viewport width, auto-scrolls to bottom on content updates, and refreshes on a timer. |
| Pane tab bar | **Done** | `h` / `l` cycles pane tabs. |
| New workspace dialog | **Done** | Name/path/branch inputs plus agent/layout selectors. |
| Path autocomplete | **Done** | Live suggestions while typing, `Tab` expands/cycles shell-style, `Enter` advances/submits, `Up`/`Down` cycle, hint line rendered. |
| Delete confirmation dialog | **Done** | Confirm delete flow exists. |
| Help overlay | **Done** | Now distinguishes available vs unavailable shortcuts. |
| Explicit unavailable-action feedback | **Done** | Pressing unavailable roadmap keys now sets a status-bar message instead of failing silently. |
| Theme support beyond default | **Not Started** | Still only `DefaultTheme()`. |

### CLI Commands

| Command | Status | Notes |
|---------|--------|-------|
| `colosseum` | **Done** | Launches the dashboard. |
| `colosseum new` | **Done** | Supports only `claude` and `codex`; validates layout and agent support. |
| `colosseum list` | **Done** | Refreshes live statuses before output. |
| `colosseum attach` | **Done** | Switches into the workspace session and installs a `prefix+e` return binding back to the dashboard session you launched from. |
| `colosseum delete` | **Done** | Works. |
| `colosseum broadcast` | **Not Started** | Not present. |
| `colosseum diff` | **Not Started** | Not present. |

### Keybindings

| Key | Action | Status | Notes |
|-----|--------|--------|-------|
| `j/k` | Navigate workspace list | **Done** | Implemented. |
| `h/l` | Switch preview pane tab | **Done** | Implemented. |
| `Enter` | Attach to workspace | **Done** | Implemented. |
| `n` | New workspace dialog | **Done** | Implemented. |
| `d` | Delete workspace | **Done** | Implemented. |
| `J` | Jump to next needing attention | **Done** | Implemented. |
| `?` | Help overlay | **Done** | Implemented. |
| `q` | Quit | **Done** | Implemented. |
| `prefix+e` | Return from attached workspace to dashboard | **Done** | Installed when Colosseum switches the tmux client into a workspace session; targets the session that launched the dashboard. |
| `b` | Broadcast prompt | **Unavailable** | No implementation yet; pressing it shows an unavailable message. |
| `D` | Diff viewer | **Unavailable** | Same behavior. |
| `r` | Rename workspace | **Unavailable** | Same behavior. |
| `/` | Filter/search workspaces | **Unavailable** | Same behavior. |
| `m` | Mark notifications read | **Unavailable** | Same behavior. |
| `R` | Restart agent | **Unavailable** | Same behavior. |
| `s` | Stop agent | **Unavailable** | Same behavior. |

### Configuration

| Feature | Status | Notes |
|---------|--------|-------|
| TOML config file | **Not Started** | No `internal/config/` package. |
| Config-driven poll interval | **Not Started** | Poll interval is still hardcoded. |
| Config-driven default agent/layout | **Not Started** | Defaults are still hardcoded. |
| Config-driven worktree settings | **Not Started** | Not implemented. |
| Config-driven notification settings | **Not Started** | Not implemented. |
| Config-driven theme name | **Not Started** | Not implemented. |

---

## Known Issues And Important Gaps

1. `UnreadCount` is still dead state. The field exists and renders, but nothing updates it.
2. YOLO flags exist in agent definitions, but there is still no code path that uses them.
3. The roadmap features are still absent: worktrees, notifications, broadcast, diffing, restart/stop behavior, rename/filter/mark-read flows, and config.
4. The project vision still references git worktrees, but the actual create path still just launches tmux sessions in an existing directory and stores branch metadata.
5. The intentional supported create surface is only `claude` and `codex`, even though legacy definitions for Gemini/OpenCode/Aider remain registered.
6. OpenCode and Aider status definitions still have no dedicated fixture coverage.
7. Status detection is materially better after the 2026-03-07 hardening (ANSI stripping, tiered windows, spike/hysteresis, pane title signal), but it is still fundamentally heuristic and regex-driven. The most impactful next step would be **hook-based detection for Claude Code** — using Claude's native `settings.json` hooks (`PreToolUse`, `UserPromptSubmit`, `Stop`, `Notification`) to have Claude write its own state to a file, like agent-of-empires and cmux do. This eliminates the regex arms race entirely for Claude. See the research notes in the 2026-03-07 session for detailed implementation patterns from both projects.
8. “Waiting” semantics are still only partially semantic. The current rule now treats prompt-plus-recent-question states as waiting, but there is still no protocol-level signal that cleanly separates “assistant asked a question in prose” from “assistant is truly blocked on user input.”
9. The app still has two status-update authorities: the background poller and the direct refresh call used during initial TUI load. This is workable now, but still worth consolidating before larger event flows are added.
10. The `PaneCapturer` interface now has two methods (`CapturePane`, `CapturePaneTitle`). This was the simplest approach — no other project in the research (cmux, AoE, hive, agent-deck, claude-squad) uses type assertions for optional tmux capabilities. Every concrete implementor needs both methods.

Things that are no longer current issues and should not be re-raised as if unfixed:

- silent no-op roadmap keybindings
- ignored agent launch errors
- stale CLI list statuses from JSON only
- invalid layout strings silently degrading to one pane
- delete ignoring every tmux kill error
- partial workspace create leaking tmux sessions on downstream failure
- preview content overflowing past the pane border
- no deterministic way back to the dashboard after attaching to a workspace
- current live Claude/Codex CLI output missing from fixture coverage

---

## Recommended Next Steps

If picking up from here, the best order is:

1. **Decide whether Colosseum is primarily a tmux dashboard or a real worktree manager**
   - The code and docs still do not fully agree.
   - This decision should drive whether `internal/worktree/` is the next architectural layer or whether the docs/product language should narrow.

2. **Hook-based detection for Claude Code (Option B from the research)**
   - Use Claude's native `settings.json` hooks to write status to a file (`/tmp/colosseum-hooks/<workspace-id>/status`).
   - Hook events: `PreToolUse` → running, `UserPromptSubmit` → running, `Stop` → idle, `Notification` (matcher: `permission_prompt|elicitation_dialog`) → waiting.
   - Read the status file in the poller before falling back to pane scraping.
   - Add a 5-minute staleness threshold (as agent-of-empires does).
   - This gives 100% accurate status for Claude with zero regex fragility. Keep pane scraping as fallback for Codex and unconfigured Claude instances.

3. **Continue status-system hardening while the surface is still small**
   - Consolidate the split refresh paths.
   - Tighten waiting semantics further if false positives remain.
   - Add a lightweight fixture-capture workflow for current agent panes before more CLIs drift.

4. **Implement notifications and unread-count plumbing**
   - The model already exposes `UnreadCount`.
   - Poller updates are already available.
   - This is the next clean layer after the recent status hardening.

5. **Implement restart/stop first among the unavailable TUI actions**
   - These are simpler than broadcast/diff/rename/filter and would convert placeholders into genuinely useful operational controls.

6. **Add config loading**
   - Poll interval, default agent, default layout, and theme selection are the obvious first values.

---

## Suggested Starting Points In The Code

- CLI bootstrap: `cmd/colosseum/main.go`
- Workspace lifecycle: `internal/workspace/manager.go`
- Persistence: `internal/workspace/storage.go`
- Live status refresh helper: `internal/status/refresh.go`
- Poller and updates: `internal/status/poller.go`
- Detection heuristics: `internal/status/detector.go`
- ANSI normalization: `internal/status/normalize.go`
- Agent definitions: `internal/agent/*.go`
- TUI root app: `internal/tui/app.go`
- New workspace dialog: `internal/tui/dialog/new_workspace.go`
- Help overlay: `internal/tui/dialog/help.go`

---

## Validation Commands

These are the relevant commands for continuing work safely:

```bash
go test ./... -count=1
go build -o colosseum ./cmd/colosseum
```

If you change the tmux command layer or workspace lifecycle, re-run both before committing.
