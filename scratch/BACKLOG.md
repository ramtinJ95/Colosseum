# BACKLOG

- Avoid treating unsent composer text as agent errors in status detection. [`internal/status/detector.go`](/home/ramtinj/personal-workspace/Colosseum/internal/status/detector.go) checks `ErrorPatterns` against all recent non-empty pane content once the bottom line stops looking like a bare prompt, and [`internal/agent/patterns.go`](/home/ramtinj/personal-workspace/Colosseum/internal/agent/patterns.go) uses broad matches like `rate limit`, `fatal error`, and `authentication failed`. If broadcast leaves a prompt sitting unsent in the composer, those words can come from the draft itself and falsely flip a workspace to `Error` even though the agent never executed anything.

- Update stale next-step guidance in `improvements-plan.md:299-301`. The document says Milestone A is complete at `improvements-plan.md:242-263`, but the recommended next step still says to start Milestone A instead of Milestone B.
