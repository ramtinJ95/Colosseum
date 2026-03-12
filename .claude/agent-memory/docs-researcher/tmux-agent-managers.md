# Tmux Agent Manager - Status Detection Research

## Date: 2026-03-07

Comprehensive research into how tmux-based AI agent managers detect agent states.
Covers 4 projects with 3 fundamentally different approaches.

---

## Approach 1: Pane Content Hashing + String Matching (claude-squad)

**Repo:** `smtg-ai/claude-squad` (Go, AGPL-3.0)
**Key files:** `session/instance.go`, `session/tmux/tmux.go`, `app/app.go`

### State Model
```go
type Status int
const (
    Running Status = iota  // agent is working
    Ready                  // waiting for user input
    Loading                // starting up
    Paused                 // worktree removed, branch preserved
)
```

### Core Algorithm (in `app/app.go` tick loop)
- Polls every ~100ms via `tickUpdateMetadataCmd`
- For each instance: calls `HasUpdated()` which returns `(updated bool, hasPrompt bool)`
- If content changed (hash differs): set `Running`
- If content unchanged AND has permission prompt: auto-tap Enter (if AutoYes)
- If content unchanged AND no prompt: set `Ready`

### Detection Mechanism (`session/tmux/tmux.go`)
- Uses `tmux capture-pane -p -e -J` to grab terminal content
- SHA256 hash of full content compared to previous hash
- **No content normalization** -- raw hash comparison
- Prompt detection via simple `strings.Contains`:
  - Claude: `"No, and tell Claude what to do differently"`
  - Aider: `"(Y)es/(N)o/(D)on't ask again"`
  - Gemini: `"Yes, allow once"`
- Trust prompt: `"Do you trust the files in this folder?"`

### Weakness
- No normalization means spinner animations, token counters, timing info
  all cause hash changes, so agent always appears "Running" during processing
  (which is correct but indistinguishable from "just changed output once")
- Only distinguishes Running vs Ready -- no "needs approval" separate state

---

## Approach 2: Hook-Based State Notification (tmux-agent-indicator)

**Repo:** `accessd/tmux-agent-indicator` (Bash)
**Key files:** `scripts/agent-state.sh`, `hooks/claude-hooks.json`, `adapters/codex-notify.sh`

### State Model
```
running | needs-input | done | off
```

### Core Algorithm
- **Does NOT scrape pane content at all**
- Installs hooks directly into agent config files:
  - Claude: `~/.claude/settings.json` hooks for `UserPromptSubmit`, `PermissionRequest`, `Stop`
  - Codex: `~/.codex/config.toml` notify command
  - OpenCode: JS plugin in `~/.config/opencode/plugins/`
- Hooks call `agent-state.sh --agent <name> --state <state>`
- State stored in tmux environment variables: `TMUX_AGENT_PANE_${pane_id}_STATE`

### Claude Hook Events
- `UserPromptSubmit` -> first sets `off` (reset), then sets `running`
- `PermissionRequest` -> sets `needs-input`
- `Stop` -> sets `done`

### Codex Adapter (`codex-notify.sh`)
Maps codex events to states:
- `start|session-start|turn-start|working` -> `running`
- `permission*|approve*|needs-input|input-required|ask-user` -> `needs-input`
- `agent-turn-complete|complete|done|stop|error|fail*` -> `done`

### Strengths
- Zero polling overhead -- event-driven
- 100% accurate (agent itself reports state)
- Supports visual indicators: pane borders, window title colors, status bar icons, animations

### Weakness
- Requires installing hooks into each agent's config
- Only works with agents that support hooks/notifications

---

## Approach 3: Deep Content Analysis + Spike Detection (colonyops/hive)

**Repo:** `colonyops/hive` (Go)
**Key files:** `internal/core/terminal/detector.go`, `internal/core/terminal/state_tracker.go`, `internal/core/terminal/tmux/tmux.go`

### State Model
```go
type Status string
const (
    StatusActive   Status = "active"   // spinner/busy indicator
    StatusApproval Status = "approval" // permission dialog (HIGH URGENCY)
    StatusReady    Status = "ready"    // input prompt visible
    StatusMissing  Status = "missing"  // session not found
)
```

### Architecture
Three-layer detection system:
1. **Detector** -- pattern matching on terminal content
2. **StateTracker** -- spike detection + hysteresis filtering
3. **tmux Integration** -- caching, rate limiting, multi-window discovery

### Detector: `IsBusy(content)`
Checks last 15 non-empty lines for:
1. Explicit busy text: `"ctrl+c to interrupt"`, `"esc to interrupt"`
2. Spinner characters (skipping box-drawing lines):
   - Braille: `"...", "...", "...", "...", "...", "...", "...", "...", "...", "..."`
   - Asterisk (Claude 2.1.25+): `"...", "...", "...", "..."`
3. 90+ whimsical words with ellipsis: `"pondering..."`, `"clauding..."`, etc.
4. Timing patterns: `"thinking"` + `"tokens"`, `"connecting"` + `"tokens"`

### Detector: `NeedsApproval(content)`
Priority: returns false if IsBusy. Checks last 15 lines for:
- Permission prompts: "No, and tell Claude what to do differently", "Yes, allow once", etc.
- Box-drawing dialogs: "| Do you want", "| Allow"
- Selection indicators: "... Yes", "... No"
- Confirmation patterns: "(Y/n)", "[Y/n]", "Continue?", "Proceed?"
- MCP/tool prompts: "Allow this MCP server", "Run this command?"
- Interactive: "Use arrow keys to navigate", "Press Enter to select"
- Codex-specific: "Would you like to run the following command?"

### Detector: `IsReady(content)`
Priority: returns false if IsBusy or NeedsApproval. Checks last 5 lines for:
- Standalone prompt: `">"`, `"..."` (with ANSI stripping + NBSP normalization)
- Prompt with suggestion: `"... Try "`
- Codex: `"codex>"`, `"Continue?"`, `"How can I help"`

### StateTracker: Spike Detection
- **SpikeWindow**: 1 second -- requires sustained activity to confirm state change
- **HysteresisWindow**: 500ms -- minimum time to hold a status before allowing change
- Exception: `StatusApproval` always transitions immediately (user is waiting)
- Tracks `lastStableStatus` and `lastStatusTime` for filtering

### Content Normalization (for hash-based change detection)
Strips before hashing:
- ANSI escape codes (O(n) single-pass parser)
- Spinner characters (braille + asterisk)
- Dynamic status counters: `"(45s . 1234 tokens)"` -> `"(STATUS)"`
- Thinking patterns: `"... Gusting... (35s . ... 673 tokens)"` -> `"THINKING..."`
- Progress bars, percentages, download progress
- Time patterns, status line patterns, git branch, beads count
- Multiple blank lines collapsed
- Trailing whitespace per line

### tmux Integration Layer
- `RefreshCache()` calls `tmux list-windows -a -F "#{session_name}\t#{window_index}\t..."`
- Per-window `RateLimiter` (max 2 captures/second)
- Only re-captures pane content when `window_activity` timestamp changes
- Preferred window patterns (regex) for multi-window sessions

---

## Approach 4: Hybrid Pane Content + Title Detection (agent-deck)

**Repo:** `asheshgoplani/agent-deck` (Go)
**Key files:** `internal/tmux/detector.go`, `internal/tmux/title_detection.go`

### State Model
```go
type SessionState string
const (
    StateIdle    SessionState = "idle"    // waiting for user
    StateBusy    SessionState = "busy"    // actively working
    StateWaiting SessionState = "waiting" // needs input
)
```

### Dual Detection: Title + Content

#### Title Detection (`title_detection.go`)
Claude Code sets pane titles via OSC escape sequences:
- **Braille spinner** (U+2800-U+28FF) in title = `TitleStateWorking`
- **Done markers** (stars, asterisk) in title = `TitleStateDone`
- Otherwise = `TitleStateUnknown`

Uses `tmux list-panes -a -F "#{session_name}\t#{pane_title}\t#{pane_current_command}\t..."`
Caches results with 4-second TTL, refreshed once per tick.

#### Content Detection (`detector.go`)
`PromptDetector.HasPrompt(content)` -- tool-specific, very detailed:

**Claude detection priority:**
1. BUSY indicators (return false if found):
   - `"ctrl+c to interrupt"`, `"esc to interrupt"`
   - Spinner chars in last 10 lines (same braille + asterisk set, skipping box-drawing)
   - Whimsical word + unicode ellipsis + "tokens" pattern
   - `"thinking"` + `"tokens"`, `"connecting"` + `"tokens"`
2. WAITING - Permission prompts (same comprehensive list as hive)
3. WAITING - Input prompt: `">"` or `"..."` in last line (ANSI stripped, NBSP normalized)
4. WAITING - Prompt in last 5 lines (handles status bar after prompt)
5. WAITING - Question prompts: "Continue?", "(Y/n)", etc.
6. WAITING - Completion + prompt combo: "Task completed" + ">" nearby

**OpenCode detection:**
- Busy: `"esc interrupt"`, pulse spinner chars, task text ("Thinking...", "Generating...")
- Ready: `"press enter to send"`, `"Ask anything"`, line ending with ">"

**Gemini detection:**
- `"gemini>"`, `"Yes, allow once"`, `"Type your message"`, line ending with ">"

**Codex detection:**
- Busy: `"esc to interrupt"`, `"ctrl+c to interrupt"`
- Ready: `"codex>"`, `"Continue?"`

---

## Summary: Detection Approaches Compared

| Feature | claude-squad | tmux-agent-indicator | hive | agent-deck |
|---|---|---|---|---|
| Method | Pane hash + string match | Hook-based events | Deep content analysis | Title + content analysis |
| Polling | Yes (~100ms) | No (event-driven) | Yes (configurable) | Yes (per-tick) |
| Content normalization | None | N/A | Extensive (regex) | None (ANSI strip only) |
| Spike detection | None | N/A | Yes (1s window + 500ms hysteresis) | None |
| States | 4 (Running/Ready/Loading/Paused) | 4 (running/needs-input/done/off) | 4 (active/approval/ready/missing) | 3 (idle/busy/waiting) |
| Approval vs Ready | Combined | Separate | Separate (approval=high urgency) | Separate |
| Multi-agent | Per-agent detection | Per-agent hooks | Per-window detection | Per-session detection |
| Rate limiting | None | N/A | 2 captures/sec/window | Cached per tick |
| Title-based | No | No | No | Yes (OSC braille/done markers) |
