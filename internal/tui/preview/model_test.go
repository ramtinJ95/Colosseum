package preview

import (
	"strings"
	"testing"
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
