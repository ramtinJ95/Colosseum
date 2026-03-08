# Repo-Centric UI

Later-phase design note for evolving Colosseum from a workspace-first interface into a repository-first interface once the worktree foundation and experiment model already exist.

This document is intentionally scoped to a later phase. It should not block landing worktree support with the current UI.

## Purpose

The current UI is workspace-centric:

- the sidebar lists workspaces
- each workspace is mainly a tmux session plus agent runtime
- actions are targeted at workspaces directly

That is a good shape for the current product, but compare/vote and multi-candidate workflows will eventually benefit from a higher-level structure:

- repository
- experiment or standalone checkout
- workspace attached to that checkout

The repo-centric UI is the later visible evolution of that model.

## Why A Repo-Centric UI Exists

When several worktrees belong to the same repository, the user usually wants to reason about:

- which repo they belong to
- which checkouts are related
- which candidates belong to the same experiment
- which workspaces are currently active
- which candidate won and should be merged

In those flows, workspaces are not the main object. They are runtimes attached to a code candidate.

## Design Principle

The UI should eventually reflect the real conceptual hierarchy:

1. `Repository`
2. `Checkout` or `Experiment`
3. `Workspace`

This is a display-layer evolution, not a separate data model. The underlying repository and checkout-aware model should already exist before this UI is built.

## What Changes Visibly

### Current shape

The user sees something like:

- `auth-fix-claude`
- `auth-fix-codex`
- `payments-refactor`
- `docs-cleanup`

All of these appear as equally flat workspace entries.

### Repo-centric shape

The user would instead see something like:

- `myrepo`
  - experiment: `Auth fix comparison`
    - candidate checkout: `Auth fix Claude`
    - candidate checkout: `Auth fix Codex`
  - standalone checkout: `Payments refactor`
  - standalone checkout: `Docs cleanup`

Active tmux workspaces are shown as runtimes attached to those entries, not as the only visible entities.

## UI Goals

- Make it easy to understand which candidates belong together.
- Make it easy to compare candidates from the same experiment.
- Make standalone worktrees remain simple and not feel over-modeled.
- Preserve fast navigation for the current power-user workflow.
- Keep active tmux runtime state visible without making tmux the only organizing principle.

## Proposed Information Architecture

### Level 1: Repository

Each repository entry should summarize:

- repository name
- root path
- default branch
- number of standalone checkouts
- number of experiments
- number of active workspaces

Possible repository actions:

- new standalone workspace
- new standalone worktree
- new experiment
- prune finished checkouts later

### Level 2A: Experiment

An experiment groups related candidate checkouts created for one prompt or task.

Each experiment entry should summarize:

- experiment title
- prompt or prompt summary
- base branch
- candidate count
- active workspace count
- evaluation status
- winner if selected

Possible experiment actions:

- broadcast prompt to all candidates
- open compare view
- vote/select winner
- merge winner
- remove losing candidates

### Level 2B: Standalone Checkout

A standalone checkout represents unrelated work that does not belong to an experiment.

Each standalone checkout entry should summarize:

- title
- branch
- base branch
- active workspace state
- worktree ownership

Possible checkout actions:

- attach/open workspace
- open diff
- merge
- remove

### Level 3: Workspace Runtime

If a checkout has one or more active workspaces, those runtimes should show:

- workspace title
- agent type
- status
- pane layout
- session state

This keeps tmux runtime visible without making it the top-level grouping.

## View Concepts

### Sidebar Structure

Possible later sidebar structure:

- repositories
  - experiments
    - candidate checkouts
  - standalone checkouts

Collapsed and expanded states should be supported.

### Header

When an item is selected, the header should show the full context:

- repository
- experiment title if applicable
- checkout branch
- workspace title if attached

### Preview Panel

Preview remains useful, but should become context-aware.

Depending on selection:

- repository selected: summary view
- experiment selected: candidate table and status summary
- checkout selected: Git metadata, diff summary, winner state later
- workspace selected: current pane preview as today

## Compare/Vote UX Direction

The repo-centric UI exists mainly to make compare/vote feel natural.

### Compare entry points

- from an experiment
- from a multi-select of arbitrary checkouts

### Compare targets

- base branch vs one checkout
- checkout A vs checkout B
- all candidates in one experiment

### Vote/evaluation outcomes

The UI should eventually support:

- mark winner manually
- record notes
- show why a winner was chosen
- separate evaluation from merge

Important rule:

- voting is not merging
- selecting a winner should be reversible until merge/cleanup is explicitly run

## Why This Is Deferred

This UI should wait until the underlying model already exists because otherwise the visible structure will fight the implementation.

It is deliberately deferred because:

- worktree lifecycle is the more urgent missing capability
- the current workspace-first UI is already functional
- compare/vote can start on top of the current UI with the right model underneath
- moving the whole visible product shape too early would increase scope sharply

## Migration Strategy

Recommended order:

1. land repository and checkout-aware persistence
2. land worktrunk-backed worktree lifecycle
3. land experiment grouping
4. land compare/vote actions on top of the existing UI
5. only then, if it clearly improves usability, evolve the visible UI into repo-centric grouping

This preserves velocity while keeping the long-term direction open.

## Open Questions For The Later Phase

- Should repositories be the only top-level sidebar nodes, or should users be able to keep a flat workspace view as an alternate mode?
- Should experiments appear inline with standalone checkouts, or in a dedicated section under each repository?
- Should compare/vote live in the main preview pane, a dedicated full-screen view, or a modal workflow?
- Should users be able to pin favorite repositories or experiments?
- Should repository nodes show only Colosseum-known checkouts, or all discoverable worktrunk checkouts?

## Design Summary

The repo-centric UI is the later visual form of a model that should be introduced earlier.

Short version:

- current UI stays workspace-first for now
- later UI becomes repository-first
- experiments and checkouts become visible groupings
- workspaces become attached runtimes instead of the only visible entities

That is the future shape once worktree orchestration and compare/vote are already working well enough to justify a larger UI evolution.
