package sidebar

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/agent"
	apptheme "github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestUpdateUsesConfiguredNavigationBindings(t *testing.T) {
	model := New().WithNavigationKeys(
		key.NewBinding(key.WithKeys("w")),
		key.NewBinding(key.WithKeys("s")),
	)
	model.SetWorkspaces([]workspace.Workspace{
		{ID: "ws-1", Title: "one", AgentType: agent.Claude},
		{ID: "ws-2", Title: "two", AgentType: agent.Codex},
	})

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if updated.Cursor != 1 {
		t.Fatalf("cursor after down key = %d, want 1", updated.Cursor)
	}

	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if updated.Cursor != 0 {
		t.Fatalf("cursor after up key = %d, want 0", updated.Cursor)
	}
}

func TestWithThemeOverridesDefaultTheme(t *testing.T) {
	custom := apptheme.DefaultTheme()
	custom.AppTitle = custom.AppTitle.Foreground(lipgloss.Color("201"))

	model := New().WithTheme(custom)

	if !sameColor(model.theme.AppTitle.GetForeground(), custom.AppTitle.GetForeground()) {
		t.Fatal("sidebar theme foreground did not use configured theme")
	}
}

func sameColor(a, b interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
