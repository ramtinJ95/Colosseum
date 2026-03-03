# Colosseum

Go TUI for managing parallel AI coding agents across git worktrees, built on tmux and Bubble Tea.

## Build & Test

```bash
make build    # compile to ./colosseum
make test     # run all tests
make vet      # go vet ./...
make lint     # golangci-lint (if installed)
```

## Project Conventions

- **Go 1.24+** with modules
- All internal packages under `internal/`
- Tests use the standard `testing` package; no external test frameworks
- Fixture-driven tests for status detection in `testdata/fixtures/`
- Interfaces defined in the consumer package, not the provider
- Error types use `fmt.Errorf` with `%w` wrapping
- tmux session prefix: `colo-`
- JSON state file: `~/.config/colosseum/workspaces.json`
- TOML config: `~/.config/colosseum/config.toml`

## Architecture

- `internal/tmux/` — tmux command abstraction (os/exec, no library)
- `internal/agent/` — agent type definitions and detection patterns
- `internal/workspace/` — workspace model, persistence, lifecycle
- `internal/status/` — background status detection engine
- `internal/tui/` — Bubble Tea TUI (sidebar, preview, theme)
- `cmd/colosseum/` — CLI entry point (cobra)

## Key Interfaces

| Interface | Package | Purpose |
|-----------|---------|---------|
| `tmux.Commander` | tmux | os/exec abstraction for testing |
| `workspace.SessionCreator` | workspace | tmux session creation |
| `status.PaneCapturer` | status | tmux pane capture |
| `status.WorkspaceProvider` | status | workspace listing |
