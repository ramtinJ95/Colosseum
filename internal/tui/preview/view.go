package preview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

func (m Model) View() string {
	t := theme.DefaultTheme()

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

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	var parts []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			parts = append(parts, activeStyle.Render(fmt.Sprintf("[%s]", tab)))
		} else {
			parts = append(parts, inactiveStyle.Render(tab))
		}
	}

	return " " + strings.Join(parts, " ") + "  " + t.Dim.Render("h/l: switch pane")
}
