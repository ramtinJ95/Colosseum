package preview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	apptheme "github.com/ramtinj/colosseum/internal/tui/theme"
)

func TestSetContentWrapsLongLinesToViewportWidth(t *testing.T) {
	model := New()
	model.SetSize(20, 10)
	model.SetContent("test", "this is a deliberately long line that should wrap inside the preview")

	view := model.viewport.View()
	for _, line := range strings.Split(view, "\n") {
		if lineWidth := len([]rune(line)); lineWidth > model.viewport.Width {
			t.Fatalf("line length %d exceeds viewport width %d: %q", lineWidth, model.viewport.Width, line)
		}
	}
}

func TestWithThemeOverridesDefaultTheme(t *testing.T) {
	custom := apptheme.DefaultTheme()
	custom.ActiveTab = custom.ActiveTab.Foreground(lipgloss.Color("201"))

	model := New().WithTheme(custom)

	if !sameColor(model.theme.ActiveTab.GetForeground(), custom.ActiveTab.GetForeground()) {
		t.Fatal("preview theme foreground did not use configured theme")
	}
}

func sameColor(a, b interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
