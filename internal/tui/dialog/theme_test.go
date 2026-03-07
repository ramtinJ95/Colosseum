package dialog

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	apptheme "github.com/ramtinj/colosseum/internal/tui/theme"
)

func TestWithThemeOverridesDefaultTheme(t *testing.T) {
	custom := apptheme.DefaultTheme()
	custom.DialogBorder = custom.DialogBorder.BorderForeground(lipgloss.Color("201"))

	newWorkspace := NewNewWorkspace().WithTheme(custom)
	deleteDialog := NewDelete("ws-1", "demo").WithTheme(custom)
	helpDialog := NewHelp().WithTheme(custom)

	if !sameColor(newWorkspace.theme.DialogBorder.GetBorderTopForeground(), custom.DialogBorder.GetBorderTopForeground()) {
		t.Fatal("new workspace dialog did not keep configured theme")
	}
	if !sameColor(deleteDialog.theme.DialogBorder.GetBorderTopForeground(), custom.DialogBorder.GetBorderTopForeground()) {
		t.Fatal("delete dialog did not keep configured theme")
	}
	if !sameColor(helpDialog.theme.DialogBorder.GetBorderTopForeground(), custom.DialogBorder.GetBorderTopForeground()) {
		t.Fatal("help dialog did not keep configured theme")
	}
}

func sameColor(a, b interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
