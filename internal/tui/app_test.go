package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/config"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type appTestCapturer struct {
	content string
}

func (c *appTestCapturer) CapturePane(_ context.Context, _ string, _ int) (string, error) {
	return c.content, nil
}

func (c *appTestCapturer) CapturePaneTitle(_ context.Context, _ string) (string, error) {
	return "", nil
}

func TestPreviewRefreshMsgUpdatesSelectedPane(t *testing.T) {
	capturer := &appTestCapturer{content: "first output"}
	detector := status.NewDetector(capturer, 50)

	app := NewApp(nil, nil, nil, detector, config.Default())
	app.preview.SetSize(80, 20)
	app.sidebar.SetWorkspaces([]workspace.Workspace{{
		ID:        "ws-1",
		Title:     "test",
		AgentType: agent.Codex,
		PaneTargets: map[string]string{
			"agent": "%1",
		},
	}})
	app.updatePreviewContent()

	capturer.content = "second output"

	model, cmd := app.Update(previewRefreshMsg(time.Now()))
	updated := model.(App)

	if got := updated.preview.View(); !strings.Contains(got, "second output") {
		t.Fatalf("preview view = %q, want to contain %q", got, "second output")
	}
	if cmd == nil {
		t.Fatal("expected refresh to resubscribe with another command")
	}
}

func TestPreviewRefreshMsgSkipsRefreshOutsideNormalView(t *testing.T) {
	capturer := &appTestCapturer{content: "initial output"}
	detector := status.NewDetector(capturer, 50)

	app := NewApp(nil, nil, nil, detector, config.Default())
	app.preview.SetSize(80, 20)
	app.sidebar.SetWorkspaces([]workspace.Workspace{{
		ID:        "ws-1",
		Title:     "test",
		AgentType: agent.Codex,
		PaneTargets: map[string]string{
			"agent": "%1",
		},
	}})
	app.updatePreviewContent()
	app.state = viewHelp

	capturer.content = "changed output"

	model, cmd := app.Update(previewRefreshMsg(time.Now()))
	updated := model.(App)

	if got := updated.preview.View(); !strings.Contains(got, "initial output") {
		t.Fatalf("preview view = %q, want to contain %q", got, "initial output")
	}
	if cmd == nil {
		t.Fatal("expected refresh loop to continue while dialog is open")
	}
}

func TestSchedulePreviewRefreshReturnsTickCommand(t *testing.T) {
	app := NewApp(nil, nil, nil, nil, config.Default())
	cmd := app.schedulePreviewRefresh()
	if cmd == nil {
		t.Fatal("expected non-nil refresh command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("expected tick command to emit a message")
	} else if _, ok := msg.(previewRefreshMsg); !ok {
		t.Fatalf("message type = %T, want previewRefreshMsg", msg)
	}
}

func TestConfiguredSidebarKeysMoveSelection(t *testing.T) {
	cfg := config.Default()
	cfg.Keys.Up = "w"
	cfg.Keys.Down = "x"

	app := NewApp(nil, nil, nil, nil, cfg)
	app.sidebar.SetWorkspaces([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	})

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	updated := model.(App)
	if updated.sidebar.Cursor != 1 {
		t.Fatalf("cursor after configured down key = %d, want 1", updated.sidebar.Cursor)
	}

	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	updated = model.(App)
	if updated.sidebar.Cursor != 0 {
		t.Fatalf("cursor after configured up key = %d, want 0", updated.sidebar.Cursor)
	}
}
