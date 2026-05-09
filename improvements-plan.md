# Colosseum Agent-Aware Control Plane Improvements

A focused plan for borrowing the best Herdr-style ideas while keeping Colosseum tmux-native and worktree/experiment-oriented.

## Current Context

Colosseum already provides value beyond raw tmux through its workspace registry, agent launch conventions, dashboard status polling, prompt broadcast, and `worktrunk`-backed worktree/experiment lifecycle. Herdr's strongest transferable idea is not its custom PTY/terminal renderer; it is its agent-aware control plane:

1. a local API that agents and scripts can use,
2. semantic agent integrations that report state directly,
3. CLI wrappers and a skill document that teach agents how to orchestrate the tool,
4. richer status semantics such as blocked/done/seen,
5. dynamic pane/tab operations as higher-level primitives.

## Work In Progress To Consider First

These are existing repo items that may affect sequencing but do not need to block the control-plane work unless we want a cleanup pass first.

### Small correctness cleanup

Resolved in the cleanup pass before starting the control-plane work:

- Removed stale backlog entries for issues that were already fixed.
- Made CLI store initialization surface state-directory creation failures eagerly.
- Confirmed workspace rename already rolls back the tmux session when JSON persistence fails.

Remaining maintenance items in `scratch/BACKLOG.md` can be handled separately; they do not block this plan.

### Existing phase-2 roadmap overlap

`phase-2-plan.md` already lists diff/compare/merge/repo-centric UI, restart/stop, and Claude hooks. This plan should not replace it. Instead:

- This document covers the agent-aware API/control-plane track.
- `phase-2-plan.md` remains the experiment evaluation and UI roadmap.
- The two converge when semantic status and CLI/API commands make compare/evaluation workflows easier to automate.

## Guiding Principles

- Keep tmux as the runtime; do not reimplement a terminal emulator.
- Treat tmux sessions/panes as infrastructure and Colosseum workspaces/checkouts/experiments as the product model.
- Prefer semantic state reports from agent hooks when available; keep screen scraping as fallback.
- Build scriptable CLI wrappers first, then optionally add a daemon/socket if the interface stabilizes.
- Preserve privacy: logs/API responses should not persist pane content unless explicitly requested.

## Proposed Priority Order

### 1. Local API and CLI wrappers

Goal: expose existing workspace/status/tmux operations in a scriptable, agent-friendly shape.

Initial CLI surface:

```bash
colosseum workspace list --json
colosseum workspace get <id-or-title> --json
colosseum workspace create --title <title> --path <path> --agent <agent> --mode <mode>
colosseum workspace focus <id-or-title>
colosseum workspace delete <id-or-title>

colosseum pane list <workspace> --json
colosseum pane read <workspace> --pane agent --lines 80 --json
colosseum pane send <workspace> --pane agent --text "..."
colosseum pane run <workspace> --pane shell --command "go test ./..."

colosseum status get <workspace> --json
colosseum wait status <workspace> --status idle --timeout 10m
colosseum wait output <workspace> --pane agent --match "..." --timeout 30s
```

Implementation notes:

- Start as normal cobra subcommands over the existing JSON store and tmux commander.
- Add stable JSON response structs in a small package, likely `internal/api` or `internal/cliapi`.
- Resolve workspaces by ID first, then exact title.
- Reuse existing `Manager`, `Store`, `Detector`, and tmux pane capture/send code.
- Avoid a long-running daemon in the first iteration unless needed by integrations.

Deliverable:

- Machine-readable commands that agents can use from shell scripts.
- Tests for JSON output, target resolution, timeouts, and tmux-command abstraction.

### 2. Semantic agent integrations

Goal: reduce reliance on fragile pane scraping by letting agent hooks report authoritative state.

State model:

```text
reported hook state > process/liveness signal > terminal heuristic > unknown
```

Suggested API shape:

```bash
colosseum agent report \
  --workspace <id> \
  --pane agent \
  --agent claude \
  --status working \
  --source claude-hook

colosseum agent release \
  --workspace <id> \
  --pane agent \
  --agent claude \
  --source claude-hook
```

Possible statuses:

- `working`
- `waiting` or `blocked` — choose one public term and map internally
- `idle`
- `error`
- `unknown`

Implementation notes:

- Add a persisted or in-memory authority layer for reported pane state.
- If no daemon exists, hook scripts can call CLI commands directly.
- If hook latency/noise becomes a problem, introduce a Unix socket server later.
- Launch agent panes with environment variables such as:

```bash
COLOSSEUM_ENV=1
COLOSSEUM_WORKSPACE_ID=<id>
COLOSSEUM_PANE=agent
COLOSSEUM_SESSION=<tmux-session>
```

Integrations to add in order:

1. Pi extension, because its event model is clean.
2. Claude Code hooks, because waiting/permission state is valuable.
3. OpenCode plugin, because it has rich permission/session events.
4. Codex hooks, with caveats around incomplete blocked-state signals.

Deliverable:

- `colosseum integration install <agent>` and `uninstall <agent>` for supported agents.
- Status detection that prefers recent authoritative reports over heuristics.
- Fallback still works without integrations.

### 3. Agent-facing `SKILL.md`

Goal: make Colosseum usable by coding agents that are running inside Colosseum or adjacent tmux panes.

Recommended contents:

- What Colosseum is and what it manages.
- How to discover current workspace and peers.
- How to read panes safely.
- How to send prompts or commands.
- How to wait for status/output.
- How to create experiment candidates.
- Safety rules: do not delete workspaces/checkouts without user intent.

Example recipes:

```bash
colosseum workspace list --json
colosseum pane read my-workspace --pane agent --lines 80
colosseum wait status my-workspace --status idle --timeout 10m
colosseum broadcast --workspaces a,b --prompt "Implement and summarize tradeoffs"
```

Deliverable:

- Root-level `SKILL.md` plus README link.
- Keep examples aligned with actual CLI wrappers from item 1.

### 4. Done/unseen status

Goal: distinguish “agent is idle and reviewed” from “agent just finished and needs review”.

Possible model:

- Existing internal agent statuses remain mostly intact.
- Add workspace/pane metadata such as `LastSeenAt`, `LastStatusChangeAt`, or `NeedsReview`.
- Surface `Done` when a workspace transitions from `Working` or `Waiting` to `Idle` while not selected/attached.
- Mark seen when selected in dashboard, attached, or explicitly acknowledged.

Design caution:

- This overlaps with old unread/notification concepts that were intentionally removed or marked out of scope.
- Keep it review-oriented, not notification-count-oriented.
- If product direction rejects unseen state, skip this item and rely on `Idle` plus status history.

Deliverable:

- A minimal `Done`/`NeedsReview` indicator useful for experiment candidate review.
- No broad notification system unless explicitly re-scoped.

### 5. Dynamic panes/tabs through tmux

Goal: make Colosseum a higher-level tmux workspace manager without owning terminal emulation.

Potential primitives:

```bash
colosseum pane split <workspace> --from agent --direction right --role shell
colosseum pane close <workspace> --pane shell
colosseum pane rename <workspace> --pane %3 --role logs
colosseum tab create <workspace> --name tests
colosseum tab focus <workspace> --name tests
```

Implementation notes:

- Map tmux panes/windows to Colosseum roles and persisted metadata.
- Keep fixed layouts as presets, not the only possible state.
- Treat tmux windows as tabs if/when tabs become first-class.
- This should follow the API work so the abstractions are already tested via CLI.

Deliverable:

- Dynamic pane management in CLI first.
- TUI controls later.

### 6. Config and theme polish

Goal: improve daily usability after the control plane is stable.

Candidates:

- `colosseum config default`
- config validation warnings with safe fallbacks for recoverable errors
- optional config reload command
- named built-in themes
- better keybinding reference in docs/TUI help

Deliverable:

- Smaller quality-of-life PRs after core API/integration decisions settle.

## Suggested Milestones

### Milestone A: Scriptable Colosseum

- Add JSON workspace/status/pane commands.
- Add wait commands for status and output.
- Add tests around command behavior and timeouts.

### Milestone B: Hook-Reported Status

- Add report/release state plumbing.
- Add Pi and Claude integrations.
- Update status resolution to prefer semantic reports.
- Add detector tests covering precedence and fallback.

### Milestone C: Agent Self-Service

- Add `SKILL.md`.
- Add examples to README.
- Verify an agent can discover, read, wait, and report status using documented commands.

### Milestone D: Review-Oriented Status

- Decide whether `Done`/unseen fits the product.
- If yes, implement minimal needs-review semantics.
- If no, document the decision and avoid reintroducing unread/notification behavior.

### Milestone E: Dynamic tmux Primitives

- Add pane split/close/role commands.
- Consider tabs as tmux windows.
- Add TUI affordances only after CLI behavior is stable.

## Open Decisions

- Should the first API be CLI-only, or should we introduce a Unix socket immediately for hook performance?
- Should public status terminology use `waiting` to match Colosseum today or `blocked` to match Herdr/common agent language?
- Should hook-reported state be persisted in `workspaces.json` or treated as ephemeral runtime state?
- How should hooks identify the workspace: environment variable, tmux pane ID lookup, or both?
- Should `Done` be a first-class `agent.Status` or derived UI metadata from `Idle + unseen`?
- How much dynamic pane state should Colosseum own versus re-discovering from tmux each time?

## Recommended Next Step

Start Milestone A with the first vertical slice: `colosseum workspace list --json`, `colosseum pane read`, and `colosseum wait status`.
