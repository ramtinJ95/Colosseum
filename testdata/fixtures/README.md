# Fixture Refresh Workflow

Use [`scripts/capture_fixture.sh`](/home/ramtinj/personal-workspace/Colosseum/scripts/capture_fixture.sh) to capture fresh tmux pane samples when agent CLIs drift.

## Release Checklist

1. Start one workspace per supported creation agent you care about validating for the release.
2. Put each pane into the state you want to preserve, for example `working`, `waiting`, `idle`, or `error`.
3. Capture the pane into `testdata/fixtures/`:

```bash
scripts/capture_fixture.sh \
  --agent codex \
  --status working \
  --pane %3 \
  --name current_working_status_bar
```

4. If pane-title-based working detection matters for the sample, capture the title too:

```bash
scripts/capture_fixture.sh \
  --agent claude \
  --status working \
  --pane %5 \
  --name spinner \
  --capture-title
```

5. Review the captured file, rename or replace the target fixture intentionally, then rerun:

```bash
go test ./internal/status -run TestDetectFromContent -count=1
```

## Minimum Suggested Coverage Before Release

- `claude`: at least one current `working`, `waiting`, and `idle` sample
- `codex`: at least one current `working`, `waiting`, and `idle` sample
- `opencode`: capture samples whenever the surface changes or support is being expanded

OpenCode, Aider, and any future agent surface should get dedicated fixture coverage before their status rules are considered stable.
