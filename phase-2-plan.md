# Colosseum Phase 2 Roadmap

Forward-looking roadmap for Colosseum beyond the v1 foundation. Organized by workflow so each section can be pursued independently.

This document consolidates unfinished items from the original worktree plan and repo-centric UI design, plus remaining v1 gaps, into a single actionable reference.

---

## Compare & Evaluate

The experiment model, checkout grouping, and evaluation structs already exist in the data layer. This section covers building the user-facing workflows on top of them.

### Diff Viewer

Status: not started. No code exists yet.

- Full-screen overlay triggered by `D` keybinding (stub already wired in `app.go`)
- Diff computation against a configurable base branch (likely needs `sergi/go-diff` or shelling out to `git diff`)
- Side-by-side rendering with additions/deletions
- File list panel with navigation
- Hunk scrolling with configurable context lines
- `colosseum diff` CLI subcommand

Design notes:

- TUI package location: `internal/tui/diff/`
- Diff can target: base branch vs checkout, checkout A vs checkout B, or all candidates in one experiment
- Consider reusing `wt step diff` if worktrunk adds structured diff output later

### Compare View

Status: not started. Data model is ready (`Experiment`, `Evaluation`, `CheckoutIDs`, `WinnerCheckoutID`).

- Entry point from an experiment (sidebar or keybinding)
- Entry point from a multi-select of arbitrary checkouts
- Side-by-side or tabbed diff comparison of candidate outputs
- Summary view showing candidate status, branch, agent type
- Navigate between candidates without leaving compare mode

### Vote & Evaluation

Status: not started. `Evaluation` struct and `EvaluationMethod` enum exist in `workspace.go`.

- Mark a winner manually from compare view or experiment context
- Record evaluation notes
- Show evaluation status and winner in sidebar/preview
- Separate evaluation from merge — selecting a winner should be reversible until merge is explicitly run

Suggested evaluation methods (already defined in code):

- `manual` — human picks winner
- `vote` — future multi-reviewer flow
- `agent-assisted` — future LLM-based comparison

### Merge Flow

Status: `Client.Merge()` exists in `worktrunk/client.go` but is not wired through the UI or CLI.

- Merge selected checkout (or experiment winner) into target branch through worktrunk
- Update experiment and evaluation state after merge
- Optionally clean up loser branches/worktrees after merge
- CLI subcommand: `colosseum merge`

### Experiment Lifecycle Completion

Status: creation and broadcast work. Cleanup and state transitions are partial.

- Bulk cleanup: remove all losing candidates from a completed experiment
- Keep winner checkout intact after cleanup
- Transition experiment status through `running` → `awaiting-evaluation` → `completed` or `abandoned` based on user actions (partial — some transitions happen on delete already)
- Surface experiment status in sidebar or preview

---

## Workspace Actions

Small features that enhance the current workspace-first UI.

### Rename Workspace

Status: **implemented**. Keybinding `r` opens rename dialog overlay.

- Rename workspace title
- Update tmux session name to match
- Update JSON state

### Filter / Search Workspaces

Status: keybinding `/` is wired but shows "unavailable". No implementation.

- Text input that filters the sidebar workspace list
- Match against title, agent type, branch name
- Clear filter to return to full list

### Restart Agent

Status: keybinding `R` is wired but shows "unavailable". No implementation.

- Kill the agent process in the agent pane
- Re-launch the agent binary with the same flags
- Preserve pane layout and shell/logs panes

### Stop Agent

Status: keybinding `s` is wired but shows "unavailable". No implementation.

- Send interrupt (Ctrl-C or equivalent) to the agent pane
- Update workspace status to Stopped
- Allow manual restart afterward

### Reorder Workspaces

Status: not started. No keybinding defined.

- Move workspace up/down in the sidebar list
- Persist ordering in JSON state

---

## Repo-Centric UI

A later-phase evolution of the sidebar from workspace-first to repository-first. The underlying data model (Repository, Checkout, Experiment on Workspace) already supports this. This section is a display-layer change.

### Why

When several worktrees belong to the same repository, the user needs to reason about which candidates are related, which experiments they belong to, and which candidate won. Workspaces alone do not communicate that structure.

### Sidebar Restructuring

Current shape (flat):

```
auth-fix-claude
auth-fix-codex
payments-refactor
docs-cleanup
```

Target shape (hierarchical):

```
myrepo
  experiment: Auth fix comparison
    Auth fix Claude   [working]
    Auth fix Codex    [waiting]
  standalone: Payments refactor
  standalone: Docs cleanup
```

Design requirements:

- Repositories as top-level collapsible nodes
- Experiments and standalone checkouts grouped under each repository
- Active workspace status shown inline on checkouts
- Collapsed/expanded state should persist
- Fast keyboard navigation must not regress

### Context-Aware Preview

Depending on what is selected in the sidebar:

- **Repository selected** — summary view: name, root path, default branch, checkout count, experiment count, active workspace count
- **Experiment selected** — candidate table with status summary, prompt, base branch, winner if selected
- **Checkout selected** — git metadata, diff summary, evaluation state
- **Workspace selected** — current pane preview (same as today)

### Context-Aware Header

When an item is selected, the header should show the full breadcrumb:

- Repository → Experiment (if applicable) → Checkout → Workspace (if attached)

### Open Design Questions

These should be resolved before implementation begins:

- Should users be able to keep a flat workspace view as an alternate mode?
- Should experiments appear inline with standalone checkouts, or in a dedicated section under each repository?
- Should compare/vote live in the main preview pane, a full-screen view, or a modal workflow?
- Should users be able to pin favorite repositories or experiments?
- Should repository nodes show only Colosseum-known checkouts, or all discoverable worktrunk checkouts?

### Migration Strategy

Recommended order:

1. Land compare/vote actions on top of the current workspace-first UI first
2. Only then evolve the visible sidebar into repo-centric grouping
3. Keep a flat-view fallback if user testing shows the hierarchy adds friction for simple use cases

---

## Agent Scope

Near-term creation and integration work is intentionally limited to the supported set: Claude Code, Codex, OpenCode, and Pi. Gemini and Aider may remain as passive detection fixtures, but they are not roadmap targets for workspace creation or semantic integrations right now.

### YOLO Mode

Status: `YoloFlags` field exists on `AgentDef` for every agent. Not surfaced anywhere in the UI or CLI.

- Toggle in the new workspace dialog or CLI `new` command
- When enabled, append `YoloFlags` to the agent launch command
- Consider per-workspace persistence so restarts respect the setting

### Custom Instruction Injection

Status: not started. No fields or UI exist.

- Allow users to pass a custom system prompt or instruction file to the agent at launch time
- Could be per-workspace or per-experiment
- Implementation depends on how each agent CLI accepts instructions (e.g., Claude Code's `--system-prompt`, Codex's `--instructions`)

### Claude Code Hook Integration

Status: not started.

- Integrate with Claude Code's hook system for event-driven workflows
- Possible hooks: on workspace status change, on agent completion, on error

---

## Broadcast Enhancements

### Auto-Create Worktrees on Broadcast

Status: not started. Currently broadcast only targets existing workspaces.

- Generic broadcast flow that creates worktrees on demand as part of the broadcast action
- User specifies: repo, base branch, prompt, candidate count, agent(s)
- Colosseum creates N worktrees, creates N workspaces, then broadcasts
- This partially overlaps with experiment creation — consider whether it should be a shortcut for `CreateExperiment` or a distinct flow

### Copy-Ignored Files

Status: `Client.CopyIgnored()` exists in `worktrunk/client.go` but is not wired through the UI.

- After creating a worktree, optionally copy ignored files (e.g., `.env`, `node_modules`) from the main worktree
- Could be a toggle in the new workspace dialog or a post-creation step

---

## Worktrunk Operations Not Yet Wired

These methods exist on `worktrunk.Client` but are not exposed through the UI or CLI:

| Method | Status | Notes |
|--------|--------|-------|
| `Merge()` | Exists | Needs UI/CLI integration (see Merge Flow above) |
| `CopyIgnored()` | Exists | Needs UI toggle (see Broadcast Enhancements above) |
| `step diff` | Not implemented | Planned wt operation for structured diff |
| `step prune` | Not implemented | Planned wt operation for cleanup |

---

## Suggested Priority Order

This is a suggested sequencing, not a strict dependency chain:

1. **Diff viewer** — the most immediately useful missing feature; unblocks compare workflows
2. **Restart / Stop agent** — small scope, high daily-use value
3. **Compare view + vote/evaluation** — completes the experiment lifecycle
4. **Merge flow** — wiring the existing `Client.Merge()` through UI and CLI
5. **Rename + filter/search** — quality-of-life for growing workspace lists
6. **YOLO mode** — low-effort since `YoloFlags` already exist
7. **Repo-centric UI** — largest scope; should wait until compare/vote is proven
8. **Auto-create worktrees on broadcast** — convenience shortcut
9. **Custom instruction injection + Claude Code hooks** — agent-specific integrations
