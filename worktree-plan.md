# Worktree Plan

Detailed plan for adding `worktrunk`-backed worktree support to Colosseum while keeping the current workspace-first UI for the first implementation phase.

This document captures the intended data model, lifecycle, and future-proofing decisions so compare/vote can be added without redesigning core state later.

## Goals

- Make isolated Git worktrees a first-class capability for Colosseum work.
- Use `worktrunk` as the authority for branch and worktree lifecycle instead of re-implementing raw `git worktree` flows.
- Keep the current tmux-first UI shape for the first phase so the product can ship incrementally.
- Design the underlying model so these future workflows are enabled from the start:
  - broadcast one prompt to several sibling worktrees
  - compare multiple candidate solutions
  - vote/select a preferred candidate
  - merge or discard candidates after evaluation
  - also support unrelated standalone feature work in separate worktrees without any compare/vote flow

## Non-Goals For Phase 1

- Do not convert the visible UI into a repo-centric dashboard yet.
- Do not embed `worktrunk`'s own interactive picker into Bubble Tea.
- Do not re-implement worktree path template resolution inside Colosseum.
- Do not make compare/vote UI a prerequisite for landing worktree support.

## Current State Summary

Today Colosseum is workspace-centric:

- the primary object is a workspace
- a workspace is mainly a tmux session plus agent runtime
- `ProjectPath` is the effective working directory
- `Branch` is currently just stored metadata, not authoritative Git state
- create/delete operate on tmux sessions and local JSON state, not on real worktree lifecycle

That model is enough for tmux orchestration, but not enough for safe worktree ownership or future compare/vote.

## Core Decision

Colosseum should be split conceptually into two authority layers:

- `worktrunk` owns Git facts and worktree lifecycle
  - branch creation
  - base-branch selection
  - worktree path resolution
  - worktree listing and state
  - safe removal
  - merge flow
- Colosseum owns agent orchestration and interface state
  - tmux sessions
  - pane layout
  - agent launch
  - workspace status detection
  - prompt broadcast
  - compare/vote orchestration and UI later

This is the boundary that keeps the feature set coherent.

## Domain Model

The first implementation should introduce a repo/checkout-aware data model even if the UI still looks workspace-centric.

### Repository

Represents one repository root known to Colosseum.

Proposed fields:

- `ID`
- `RootPath`
- `DefaultBranch`
- `Backend`
- `WorktrunkAvailable`
- `CreatedAt`

Notes:

- `Backend` is expected to be `worktrunk` for managed worktree flows.
- `DefaultBranch` should come from Git facts, not user-entered labels.

### Checkout

Represents one concrete checkout candidate inside a repository. This is the object compare/vote will operate on later.

Proposed fields:

- `ID`
- `RepositoryID`
- `RepoRoot`
- `CheckoutPath`
- `Branch`
- `BaseBranch`
- `DefaultBranch`
- `MergeBaseSHA`
- `Backend`
- `Ownership`
- `CreatedFrom`
- `ExperimentID`
- `CreatedAt`

Field meanings:

- `CheckoutPath`: actual directory where tmux and agents run
- `Ownership`: whether Colosseum created and may delete this checkout, or whether it was attached from outside
- `CreatedFrom`: standalone flow or experiment flow
- `ExperimentID`: optional link for grouped candidate runs

Suggested enums:

- `Backend`
  - `worktrunk`
  - `external`
- `Ownership`
  - `colosseum-managed`
  - `attached`
- `CreatedFrom`
  - `standalone`
  - `experiment`

### Workspace

Represents a tmux session and agent runtime attached to a checkout.

Proposed direction:

- keep existing workspace behavior and UI affordances
- add `CheckoutID` so a workspace references a checkout instead of only a raw path

Proposed additional fields:

- `CheckoutID`
- `RepositoryID`

Important rule:

- `ProjectPath` remains the effective working directory for tmux creation so existing tmux and preview flows can continue to work
- `ProjectPath` should become equivalent to the checkout's resolved `CheckoutPath`

### Experiment

Represents a grouped parallel run of several candidate checkouts for the same task/prompt.

Proposed fields:

- `ID`
- `RepositoryID`
- `RepoRoot`
- `Title`
- `Prompt`
- `BaseBranch`
- `CheckoutIDs`
- `WorkspaceIDs`
- `Status`
- `WinnerCheckoutID`
- `CreatedAt`

Suggested statuses:

- `draft`
- `running`
- `awaiting-evaluation`
- `completed`
- `abandoned`

### Evaluation

Represents compare/vote outcomes. This can land after worktree support, but the model should reserve room for it.

Possible fields:

- `ID`
- `ExperimentID`
- `CheckoutIDs`
- `WinnerCheckoutID`
- `Method`
- `Notes`
- `CreatedAt`

Suggested methods:

- `manual`
- `vote`
- `agent-assisted`

## Why Checkout Must Be First-Class

Compare and vote do not fundamentally act on tmux workspaces. They act on isolated code candidates.

That means:

- a candidate may exist before an agent is attached
- a candidate may survive after a tmux workspace is removed
- multiple candidates may belong to one experiment
- ad hoc comparison should be possible between unrelated checkouts later

If worktree support is added as only two extra fields on `Workspace`, compare/vote will require a redesign soon after.

## Supported User Workflows

### Workflow A: Standalone Feature Workspace

User intent:

- create one workspace for one feature
- optionally create a new worktree for isolation
- no compare/vote involved

Flow:

1. User opens new workspace dialog
2. User chooses standalone mode
3. User chooses:
   - existing checkout
   - new worktree
4. Colosseum creates or attaches a `Checkout`
5. Colosseum creates a `Workspace` attached to that checkout

### Workflow B: Parallel Experiment

User intent:

- test one prompt or task in parallel
- broadcast the same prompt to several agents
- let each run in its own worktree
- compare and vote later

Flow:

1. User creates a new experiment
2. User chooses repo, base branch, prompt, and agent/workspace count
3. Colosseum generates sibling candidate checkouts
4. Colosseum creates workspaces attached to those checkouts
5. Prompt is broadcast to the selected experiment workspaces
6. Compare/vote happens later as a separate action

Important rule:

- experiment membership is optional
- worktree support must not require experiments

## Naming Strategy

Generated unique Git names should exist from the start, but user-facing titles should remain separate.

### Generated by default

For branches/worktrees:

- deterministic
- unique
- filesystem-safe
- automation-safe

Suggested shape:

- `exp-auth-fix-20260308-claude-a1`
- `feat-payments-refactor-20260308`

### Optional override

Users should be able to override names, but the default should be generated.

### User-facing labels

Do not overload branch names as the only visible label. Preserve a separate human title for the UI.

Examples:

- experiment title: `Auth fix comparison`
- workspace title: `Auth fix Claude`
- branch: `exp-auth-fix-20260308-claude-a1`

## Worktrunk Integration

## New package

Recommended package name:

- `internal/worktrunk/`

Reason:

- the backend decision is already intentional
- being concrete is simpler than pretending the first implementation is generic

If a second backend is ever needed later, the abstraction can be introduced on top of this package.

## Responsibilities

The package should wrap only the machine-relevant `wt` operations Colosseum needs:

- create or resolve worktree
- list worktrees as structured data
- remove worktrees
- merge candidate branches
- optional helper steps later such as `copy-ignored` or diff support

## Rules For Calling Worktrunk

- Prefer non-interactive commands only.
- Do not embed `wt switch` without arguments or its interactive picker into Bubble Tea.
- Do not depend on shell integration for `cd`.
- Prefer `--no-cd` and explicitly use resolved paths in Colosseum.
- Prefer `wt list --format=json` as the source of truth for paths and state.
- Use foreground removal when Colosseum needs deterministic deletion feedback.

## Planned operations

Phase 1 operations:

- `switch --create --no-cd`
- `list --format=json`
- `remove --foreground`

Planned later:

- `merge`
- `step diff`
- `step copy-ignored`
- `step prune`

## Why JSON List Output Matters

Colosseum should not parse human stdout from `wt switch` to determine the resolved worktree path.

Instead:

1. call `wt switch --create --no-cd`
2. call `wt list --format=json`
3. resolve the branch to its authoritative checkout path

This keeps the integration stable even if human-oriented output wording changes upstream.

## Creation Modes

The current UI should remain recognizable, but the create flow must become mode-aware.

Recommended modes:

- `Standalone workspace`
- `Standalone workspace with new worktree`
- `Experiment run`

This is better than a simple `Worktree: Yes/No` toggle because it maps cleanly onto future compare/vote behavior.

### Standalone workspace

Inputs:

- workspace title
- checkout path
- agent
- layout

Behavior:

- attach to existing checkout
- create checkout record
- create workspace

### Standalone workspace with new worktree

Inputs:

- workspace title
- repo root
- branch name or generated default
- optional base branch
- agent
- layout

Behavior:

1. create worktree through `worktrunk`
2. resolve authoritative checkout path
3. create checkout record
4. create workspace

### Experiment run

Inputs:

- experiment title
- repo root
- prompt
- base branch
- number of candidates
- agent selection strategy
- naming defaults and optional overrides
- layout

Behavior:

1. create experiment record
2. create N sibling checkouts through `worktrunk`
3. create N attached workspaces
4. store experiment relationships

## Deletion Semantics

Deletion must become ownership-aware.

The current delete flow assumes:

- kill tmux session
- remove JSON record

That is not enough once worktrees exist.

### Attached checkout

If the checkout was attached from outside Colosseum:

- kill tmux session
- remove workspace record
- keep checkout intact

### Colosseum-managed checkout

If Colosseum created the worktree:

- kill tmux session
- remove worktree through `worktrunk`
- remove workspace record
- keep or delete branch according to chosen policy

### Future experiment-aware cleanup

For experiment candidates later:

- keep winner
- remove losers in bulk
- optionally keep all candidates until evaluation is finalized

## Merge Semantics

Merge should be planned now even if it is not in the first patch set.

Guiding rule:

- merge acts on a checkout or experiment winner, not on a tmux workspace as such

Likely behavior later:

- merge selected checkout into target branch through `worktrunk`
- update experiment and evaluation state
- optionally clean up loser branches/worktrees

## Compare/Vote Readiness Requirements

These requirements should be satisfied in the first implementation even if no compare/vote UI lands yet.

### Grouping

Checkouts must be able to belong to an experiment group.

### Base awareness

Each checkout should know what base branch it was created from.

### Merge-base awareness

Each checkout should store or be able to resolve merge-base metadata.

### Identity separation

Human labels and automation-safe branch names must be separate.

### Detached runtime

Checkouts must remain meaningful even without an active tmux workspace.

## UI Direction For Phase 1

The visible UI should remain close to the current one:

- sidebar still presents workspaces
- preview still presents tmux pane content
- create/delete flows stay familiar

But the underlying state should be upgraded:

- workspaces attach to checkouts
- checkouts belong to repositories
- checkouts may belong to experiments

This preserves shipping momentum while avoiding a second redesign for compare/vote.

## Later Repo-Centric UI

The later-phase UI should be allowed to evolve toward:

- repositories as top-level containers
- experiments and standalone checkouts grouped under each repository
- workspaces shown as runtimes attached to those checkouts

That future is documented separately in `repo-centric-ui.md`.

## Phased Delivery

### Phase 1: Worktree Foundation

- add `internal/worktrunk/`
- add repository and checkout-aware persisted model
- keep current workspace-first UI shape
- support standalone and experiment-ready creation paths
- make deletion ownership-aware

### Phase 2: Experiment Grouping

- add experiment persistence and creation flow
- create several candidate checkouts in one action
- attach multiple workspaces
- support broadcast across one experiment

### Phase 3: Compare/Vote

- compare selected checkouts
- compare experiment candidates
- record winner
- merge or remove candidates after selection

### Phase 4: Repo-Centric UI

- optional visible UI reframe around repositories, experiments, and checkouts

## Testing Expectations

The first implementation should include tests for:

- worktrunk adapter command building and error handling
- checkout persistence and round-trip loading
- workspace creation attached to a checkout
- owned vs attached deletion behavior
- experiment record creation if included in the first schema

## Design Summary

Short version:

- keep the current UI shape for now
- change the data model now
- make checkout the first-class Git object
- make workspace the first-class tmux runtime
- make experiment optional but designed-in
- let `worktrunk` stay authoritative for Git lifecycle

That is the path that supports both standalone feature work and future compare/vote without a major model rewrite.
