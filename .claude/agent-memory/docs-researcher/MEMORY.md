# Docs Researcher Memory

## Tmux Agent Manager Research (2026-03-07)
Conducted deep research into AI agent management tools.
See `tmux-agent-managers.md` for detailed findings on state detection patterns.
See `stale-title-research.md` for pane title staleness handling across projects.
See `detection-window-research.md` for window sizing and false positive prevention.

Projects researched:
- **claude-squad** (smtg-ai/claude-squad) - Pane hash + string match, no line windowing
- **tmux-agent-indicator** (accessd/tmux-agent-indicator) - Hook-based events
- **agent-deck** (asheshgoplani/agent-deck) - 15-line window, multi-tier narrowing (10/5/3/1)
- **hive** (colonyops/hive) - 15-line window, narrows to 5 for prompts, hysteresis
- **agent-of-empires** (njbrake/agent-of-empires) - 50 lines captured, 30 non-empty analyzed, narrows to 10 for input prompts
- **cmux** (manaflow-ai/cmux) - Native macOS terminal, NOT tmux-based. No content-based status detection.

Notes:
- No project named "cove" or "cove-cli" was found for AI agent management.
