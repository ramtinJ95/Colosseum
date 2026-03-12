# Detection Window Sizing & False Positive Prevention Research (2026-03-07)

## Context
Research into how 5 projects handle detection window sizing and false positive prevention
when scanning terminal content for AI agent status. Focused on the problem of wider windows
(30 non-empty lines) causing patterns like `\?\s*$` to match stale output.

## Projects Analyzed

### 1. agent-of-empires (njbrake/agent-of-empires) - Rust
- Captures 50 lines from tmux, then filters to last 30 non-empty for analysis
- Spinners checked against FULL window (all lines), not just last 30
- "esc to interrupt" checked against last 30 lines only
- Permission prompts checked against last 30 lines only
- Input prompts (">") checked against last 10 non-empty lines only
- detect_claude_status is a STUB returning Idle (hook-based, not content-based)

### 2. hive (colonyops/hive) - Go
- Uses 15 non-empty lines as the standard window
- IsReady() further narrows to last 5 lines for prompt detection
- Uses content normalization + hash for spike detection
- 500ms hysteresis window prevents rapid flickering
- Approval status bypasses hysteresis

### 3. agent-deck (asheshgoplani/agent-deck) - Go
- Uses 15 non-empty lines as base window
- Spinners checked in last 10 of the 15
- Last-line prompt check: single last non-empty line
- Recent prompt check: last 5 of the 15
- Completion verification: last 3 of the 15
- Permission prompts: full captured content (no restriction)

### 4. claude-squad (smtg-ai/claude-squad) - Go
- Captures entire visible pane (no line limit)
- Uses hash-based change detection (SHA256), not pattern matching
- Status is Running if content hash changed, Ready if unchanged + no prompt
- Single string match for hasPrompt, checked against FULL content

### 5. cmux (manaflow-ai/cmux) - Swift macOS
- NOT a tmux agent manager. It is a native macOS terminal multiplexer (Ghostty-based).
- Does NOT do agent status detection from terminal content.
- Uses Claude Code hooks for lifecycle events, not content scraping.
