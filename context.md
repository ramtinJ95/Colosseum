# Colosseum -- Problem Space & Context

## The Problem

Developers increasingly run multiple AI coding agents (Claude Code, Codex, Gemini, Aider, OpenCode) in parallel on the same codebase. This creates three fundamental challenges:

1. **Isolation**: Multiple agents editing the same files on the same branch cause conflicts. Git worktrees solve this by giving each agent its own checkout of the repo on a separate branch, sharing the same `.git` database.

2. **Visibility**: With N agents running in N terminals, the developer has no centralized view of what each agent is doing, whether it needs attention, or whether it's finished. Context-switching between terminal tabs/windows to check status is expensive.

3. **Orchestration**: Starting agents, assigning tasks, broadcasting the same prompt to multiple agents, and comparing their outputs requires manual coordination across many terminal windows.

## The Ecosystem (as of March 2026)

Five notable projects occupy this space, each with a different philosophy:

### claude-squad (Go, AGPL-3.0, ~6,200 stars)

**Repo**: `smtg-ai/claude-squad`

The most popular tool in this space. Go-based TUI that creates isolated tmux sessions per agent with git worktree integration. Status detection uses a hash-based approach -- captures tmux pane content, computes SHA-256, and compares hashes between polls to detect changes. Also does prompt detection via regex for agent-specific patterns (e.g. "Yes, allow once" for permission prompts).

- Supports Claude Code, Codex, Gemini, Aider, and custom agents via `-p` flag
- Auto-accept mode (`-y`) for hands-off operation
- Configuration at `~/.claude-squad/config.json`
- Worktree branch names use a configurable prefix + session name + timestamp suffix

**Trade-offs**: AGPL license. Screen-scraping status detection can be brittle with terminal resize or unexpected output. No broadcast prompt feature. No diff viewer.

### Agent of Empires (Rust, MIT, ~960 stars)

**Repo**: `njbrake/agent-of-empires`, binary: `aoe`

Rust-based TUI session manager with the deepest feature set. Distinguishes itself with Docker container sandboxing (including Apple Containers), a layered configuration system (global -> profile -> repo-level `.aoe/config.toml`), and a paired terminal concept (each agent session has a companion shell for git/build/test).

- 6 agents: Claude Code, OpenCode, Vibe, Codex, Gemini, Cursor CLI
- Status detection via tmux pane scraping with per-agent heuristic pattern matching
- Diff viewer for comparing worktree branches against base
- Sound effects for state transitions
- Session groups and profiles for organizing work
- Hooks system (`on_create`, `on_launch`) with trust/review security model
- Theme support (4 built-in themes)

**Trade-offs**: Status detection is heuristic-based scraping, same brittleness as claude-squad (known issue with small terminal views). No inter-agent coordination. No broadcast prompt. Single maintainer (bus factor of 1).

### cmux -- manaflow-ai (Swift/AppKit, AGPL-3.0, ~3,600 stars)

**Repo**: `manaflow-ai/cmux`

A completely different approach: a **native macOS terminal emulator** that replaces your terminal app entirely. Built on libghostty (GPU-accelerated rendering), it provides workspaces, split panes, an embedded scriptable browser, and a notification system using OSC terminal escape sequences.

- Agent-agnostic (works with any CLI tool)
- Notification rings on panes needing attention, plus macOS desktop notifications
- Socket API at `/tmp/cmux.sock` for external automation
- CLI tool for scripting (`cmux notify`, `cmux split`, etc.)
- Reads Ghostty config for seamless migration
- Port scanning across all panes (detects dev servers)
- PR metadata display in sidebar

**Trade-offs**: macOS only (community forks for Linux/Windows are nascent). Not a session manager -- it's a terminal emulator, so it doesn't manage worktrees, agent lifecycle, or orchestration. No live process restore on relaunch. AGPL license.

### cmux -- craigsc (Bash, MIT, ~330 stars)

**Repo**: `craigsc/cmux`

The minimalist approach: a single ~1,100 line Bash function that wraps git worktrees for running multiple Claude Code instances. `cmux new <branch>` creates a worktree + launches Claude. `cmux merge` merges results back.

- Zero dependencies beyond git and claude CLI
- Setup/teardown hooks (`.cmux/setup`, `.cmux/teardown`)
- `cmux init` uses Claude itself to auto-generate project-specific setup scripts
- Cross-platform (anywhere bash runs)

**Trade-offs**: Claude-only. No TUI, no status detection, no notifications. You still need a terminal multiplexer to view multiple agents. Pure worktree management, not a session manager.

### Cove (Rust, MIT, ~3 stars)

**Repo**: `rasha-hantash/cove`

Newest and most focused: a Claude Code-only session manager that uses Claude Code's **hook system** for status detection instead of screen scraping. This is the key architectural differentiator -- hooks fire on events like `UserPromptSubmit`, `Stop`, `PreToolUse`, `PostToolUse`, producing JSONL event files that the sidebar reads.

- All sessions live as windows within a single tmux session group (`cove`)
- 3-pane layout per window: Claude (70%), sidebar, mini-shell
- Context summaries: reads Claude's conversation JSONL and calls haiku to generate 1-2 sentence descriptions of what each session is doing
- Pane ID-based event matching (robust against CWD collisions)
- Auto-renames windows on branch changes

**Trade-offs**: Claude Code only. No worktree management (detects but doesn't create). No desktop notifications. No broadcast. Very new (days old).

## Architectural Patterns & Lessons

### Status Detection: Two Schools

| Approach | Used By | Pros | Cons |
|----------|---------|------|------|
| **Screen scraping** (tmux capture-pane + regex) | claude-squad, AoE, Colosseum | Agent-agnostic, no agent cooperation needed | Brittle with terminal resize, agent UI changes, false positives from code containing trigger strings |
| **Hook-based events** (agent writes events via hooks) | Cove | Reliable, no false positives, low overhead | Agent-specific (only Claude Code has hooks), requires `cove init` setup step |

Colosseum currently ships with screen scraping for status detection and leaves Claude Code hook integration as future hardening work.

### Session Architecture: Two Models

| Model | Used By | Pros | Cons |
|-------|---------|------|------|
| **Separate tmux sessions** per workspace | claude-squad, AoE, Colosseum | Full isolation, independent layouts, survives TUI crash | More tmux sessions to manage |
| **Single session group** with windows | Cove | Simpler mental model, unified session | All windows share session settings, less isolation |

### Git Worktree Strategies

| Strategy | Used By | Description |
|----------|---------|-------------|
| Nested (`.worktrees/<branch>/`) | craigsc/cmux (default) | Inside repo, easy cleanup |
| Outer-nested (`<repo>-worktrees/<branch>/`) | craigsc/cmux (option) | Next to repo, avoids `.gitignore` issues |
| Sibling (`<repo>-<branch>/`) | craigsc/cmux (option) | Flat structure |
| Bare repo pattern | AoE (recommended) | `.bare/` + worktrees at repo root, cleanest for heavy worktree use |
| Template-based (`../{repo}-worktrees/{branch}`) | Colosseum (roadmap) | Configurable, user chooses pattern |

### Feature Matrix

| Feature | claude-squad | AoE | cmux (manaflow) | cmux (craigsc) | Cove | **Colosseum** |
|---------|:---:|:---:|:---:|:---:|:---:|:---:|
| Multi-agent support | 4+ | 6 | Any | Claude only | Claude only | **3 for new workspaces today** |
| Status detection | Scrape | Scrape | OSC escape | None | Hooks | **Scrape today, hooks planned** |
| Git worktrees | Yes | Yes | No | Yes | Detect only | **Roadmap, not shipped** |
| Broadcast prompt | No | No | No | No | No | **Yes** |
| Diff viewer | No | Yes | No | No | No | **Roadmap, not shipped** |
| Desktop notifications | No | Sound | macOS native | No | No | **notify-send** |
| Docker sandbox | No | Yes | No | No | No | No |
| Embedded browser | No | No | Yes | No | No | No |
| TUI dashboard | Yes | Yes | N/A (is terminal) | No | Sidebar | **Yes** |
| Cross-platform | Unix | Linux/macOS | macOS only | Any (bash) | Linux/macOS | **Linux (primary)** |
| Language | Go | Rust | Swift | Bash | Rust | **Go** |

## Where Colosseum Fits

Colosseum occupies a specific niche: a **Go-based, tmux-native TUI** for managing parallel agent sessions today while leaving worktree orchestration and richer comparison tooling on the roadmap:

1. **Broadcast prompt** -- the standout shipped differentiator. Colosseum can send the same prompt to multiple existing tmux-backed workspaces from one dashboard.

2. **Hybrid status detection** -- screen scraping for universal agent support + Claude Code hooks for high-fidelity detection when available. Best of both worlds.

3. **Diff viewer** -- only AoE has this among the competitors. Colosseum does not ship this yet, but it remains a logical roadmap follow-on for the broadcast-and-compare workflow.

4. **Go + Bubble Tea** -- same language choice as claude-squad (the most popular tool), benefiting from the mature charmbracelet TUI ecosystem. Unlike the Rust tools (AoE, Cove), Go's compilation speed and simpler toolchain lower contributor friction.

5. **Linux-first** -- cmux (manaflow) is macOS-only. Colosseum targets Linux as the primary platform with `notify-send` integration.

6. **Separate tmux sessions** per workspace (like claude-squad/AoE, unlike Cove) for full isolation, with the TUI running in its own dashboard session.

## Key Technical Decisions for Colosseum

- **tmux via `os/exec`**: Direct CLI calls to the tmux binary, no library wrapper. Standard Go approach used by claude-squad.
- **Bubble Tea (Elm architecture)**: Model-Update-View pattern for the TUI. Sub-models for sidebar, preview, diff, dialogs.
- **JSON state persistence**: Workspace state at `~/.config/colosseum/workspaces.json`. Simple, human-readable, easy to debug.
- **TOML config**: User preferences at `~/.config/colosseum/config.toml`.
- **Fixture-driven testing**: Real captured pane content in `testdata/fixtures/` tested against detection functions. Same approach as AoE.
- **Per-agent pattern registry**: Each agent type defines its own detection patterns (working, waiting, idle, error). Extensible for new agents.
- **Notification on state transitions**: Not continuous scraping for notifications, but event-driven -- only fires when status changes (Working -> Waiting, Working -> Idle, etc.).
