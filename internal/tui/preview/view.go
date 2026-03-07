package preview

import (
	"fmt"
	"strings"

	"github.com/ramtinj/colosseum/internal/tui/theme"
)

func (m Model) View() string {
	t := m.theme

	title := t.PreviewTitle.Render(m.title)
	if m.title == "" {
		title = t.Dim.Render(" No workspace selected")
	}

	tabBar := m.renderTabs(t)

	var content string
	if tabBar != "" {
		content = title + "\n" + tabBar + "\n" + m.viewport.View()
	} else {
		content = title + "\n" + m.viewport.View()
	}
	return t.PreviewBorder.Width(m.Width).Height(m.Height).Render(content)
}

func (m Model) renderTabs(t theme.Theme) string {
	if len(m.tabs) <= 1 {
		return ""
	}

	var parts []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			parts = append(parts, t.ActiveTab.Render(fmt.Sprintf("[%s]", tab)))
		} else {
			parts = append(parts, t.InactiveTab.Render(tab))
		}
	}

	return " " + strings.Join(parts, " ") + "  " + t.Dim.Render("h/l: switch pane")
}
