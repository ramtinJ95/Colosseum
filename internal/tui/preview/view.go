package preview

import (
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

func (m Model) View() string {
	t := theme.DefaultTheme()

	title := t.PreviewTitle.Render(m.title)
	if m.title == "" {
		title = t.Dim.Render(" No workspace selected")
	}

	content := title + "\n" + m.viewport.View()
	return t.PreviewBorder.Width(m.Width).Height(m.Height).Render(content)
}
