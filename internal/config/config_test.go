package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultReturnsValidConfig(t *testing.T) {
	cfg := Default()

	if cfg.Defaults.Agent != "claude" {
		t.Errorf("default agent = %q, want %q", cfg.Defaults.Agent, "claude")
	}
	if cfg.Defaults.Layout != "agent-shell" {
		t.Errorf("default layout = %q, want %q", cfg.Defaults.Layout, "agent-shell")
	}
	if cfg.Status.PollIntervalMS != 1500 {
		t.Errorf("default poll interval = %d, want 1500", cfg.Status.PollIntervalMS)
	}
	if cfg.Tmux.SessionPrefix != "colo-" {
		t.Errorf("default session prefix = %q, want %q", cfg.Tmux.SessionPrefix, "colo-")
	}
	if cfg.Keys.Quit != "q" {
		t.Errorf("default quit key = %q, want %q", cfg.Keys.Quit, "q")
	}
	if cfg.Theme.Working != "82" {
		t.Errorf("default working color = %q, want %q", cfg.Theme.Working, "82")
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}

	defaults := Default()
	if cfg.Defaults.Agent != defaults.Defaults.Agent {
		t.Errorf("agent = %q, want default %q", cfg.Defaults.Agent, defaults.Defaults.Agent)
	}
	if cfg.Status.PollIntervalMS != defaults.Status.PollIntervalMS {
		t.Errorf("poll interval = %d, want default %d", cfg.Status.PollIntervalMS, defaults.Status.PollIntervalMS)
	}
}

func TestLoadPartialConfigPreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `[defaults]
agent = "codex"

[keys]
quit = "x"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Defaults.Agent != "codex" {
		t.Errorf("agent = %q, want %q", cfg.Defaults.Agent, "codex")
	}
	if cfg.Keys.Quit != "x" {
		t.Errorf("quit key = %q, want %q", cfg.Keys.Quit, "x")
	}

	// Unset fields should retain defaults
	if cfg.Defaults.Layout != "agent-shell" {
		t.Errorf("layout = %q, want default %q", cfg.Defaults.Layout, "agent-shell")
	}
	if cfg.Status.PollIntervalMS != 1500 {
		t.Errorf("poll interval = %d, want default 1500", cfg.Status.PollIntervalMS)
	}
	if cfg.Keys.Help != "?" {
		t.Errorf("help key = %q, want default %q", cfg.Keys.Help, "?")
	}
	if cfg.Theme.Working != "82" {
		t.Errorf("working color = %q, want default %q", cfg.Theme.Working, "82")
	}
}

func TestLoadFullConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `[defaults]
agent = "codex"
layout = "agent"

[status]
poll_interval_ms = 2000
capture_lines = 100

[ui]
preview_refresh_ms = 500
sidebar_min_width = 25
sidebar_max_width = 60

[tmux]
session_prefix = "ws-"
return_key = "g"

[keys]
up = "i"
down = "k"
quit = "x"

[theme]
working = "46"
border = "100"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Defaults.Agent != "codex" {
		t.Errorf("agent = %q, want %q", cfg.Defaults.Agent, "codex")
	}
	if cfg.Defaults.Layout != "agent" {
		t.Errorf("layout = %q, want %q", cfg.Defaults.Layout, "agent")
	}
	if cfg.Status.PollIntervalMS != 2000 {
		t.Errorf("poll interval = %d, want 2000", cfg.Status.PollIntervalMS)
	}
	if cfg.Status.CaptureLines != 100 {
		t.Errorf("capture lines = %d, want 100", cfg.Status.CaptureLines)
	}
	if cfg.UI.PreviewRefreshMS != 500 {
		t.Errorf("preview refresh = %d, want 500", cfg.UI.PreviewRefreshMS)
	}
	if cfg.UI.SidebarMinWidth != 25 {
		t.Errorf("sidebar min = %d, want 25", cfg.UI.SidebarMinWidth)
	}
	if cfg.UI.SidebarMaxWidth != 60 {
		t.Errorf("sidebar max = %d, want 60", cfg.UI.SidebarMaxWidth)
	}
	if cfg.Tmux.SessionPrefix != "ws-" {
		t.Errorf("session prefix = %q, want %q", cfg.Tmux.SessionPrefix, "ws-")
	}
	if cfg.Tmux.ReturnKey != "g" {
		t.Errorf("return key = %q, want %q", cfg.Tmux.ReturnKey, "g")
	}
	if cfg.Keys.Up != "i" {
		t.Errorf("up key = %q, want %q", cfg.Keys.Up, "i")
	}
	if cfg.Keys.Down != "k" {
		t.Errorf("down key = %q, want %q", cfg.Keys.Down, "k")
	}
	if cfg.Theme.Working != "46" {
		t.Errorf("working color = %q, want %q", cfg.Theme.Working, "46")
	}
	if cfg.Theme.Border != "100" {
		t.Errorf("border color = %q, want %q", cfg.Theme.Border, "100")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte("not valid toml [[["), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}
}

func TestLoadRejectsDuplicateKeyBindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `[keys]
up = "w"
new = "w"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected duplicate key binding error, got nil")
	}
	if got := err.Error(); got != `validating config `+path+`: duplicate key binding "w" assigned to keys.up and keys.new` {
		t.Fatalf("error = %q", got)
	}
}
