# Stale Pane Title Research (2026-03-07)

## Context
Research into how projects handle stale tmux pane titles when detecting AI agent state.
Specifically: Claude Code sets braille spinner chars in pane title while working,
but the title may NOT be cleared when Claude finishes or crashes -- causing
permanent false "Working" status if title is used as a signal.

## Projects Investigated

### 1. cmux (manaflow-ai/cmux) - Does NOT use pane titles for status
- Native macOS terminal app (Swift), NOT tmux-based
- Uses Claude Code hooks (lifecycle events) for status detection
- Hook subcommands: session-start, stop, idle, notification, prompt-submit
- On stop: calls `clearClaudeStatus` via socket to clear workspace status
- Session records have `startedAt`/`updatedAt` timestamps
- TTL: `maxStateAgeSeconds = 7 days` -- prunes expired session records
- `consume()` removes session record on stop, preventing stale state

### 2. agent-of-empires (njbrake/agent-of-empires) - Pure pane content, no title
- Rust tmux manager, uses `capture_pane(50)` for 50 lines of content
- detect_claude_status checks: spinner chars, "esc to interrupt", prompt markers
- NO pane title reading at all -- purely content-based
- NO staleness tracking, TTL, or temporal comparison
- Point-in-time check every 500ms
- Crash handling: `is_pane_dead()` via `#{pane_dead}` tmux format
- If pane dead: status = Error. If idle + custom command + not dead: Unknown

### 3. colonyops/hive - DOES NOT EXIST as a public GitHub repository

## Key Takeaway
No project examined tracks pane title changes over time or implements
TTL/staleness checks on terminal title signals. The two approaches found:
1. Event-driven hooks (cmux) -- avoids the stale problem entirely
2. Pane content scraping (agent-of-empires) -- ignores title, re-scrapes content each poll
