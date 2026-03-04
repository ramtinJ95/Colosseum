package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

func (m Model) View() string {
	t := theme.DefaultTheme()

	if len(m.Workspaces) == 0 {
		empty := t.Dim.Render("  No workspaces yet.\n  Press 'n' to create one.")
		return t.SidebarBorder.Width(m.Width).Height(m.Height).Render(
			t.AppTitle.Render(" WORKSPACES") + "\n\n" + empty,
		)
	}

	var b strings.Builder
	b.WriteString(t.AppTitle.Render(" WORKSPACES"))
	b.WriteString("\n\n")

	for i, ws := range m.Workspaces {
		icon := theme.StatusIcon(ws.Status)
		styledIcon := t.StatusStyle(ws.Status).Render(icon)

		title := ws.Title
		agentStr := t.AgentName.Render(string(ws.AgentType))
		statusStr := t.StatusStyle(ws.Status).Render(ws.Status.String())
		branchStr := ""
		if ws.Branch != "" {
			branchStr = t.BranchName.Render(fmt.Sprintf("[%s]", ws.Branch))
		}

		unread := ""
		if ws.UnreadCount > 0 {
			unread = t.UnreadBadge.Render(fmt.Sprintf(" (%d)", ws.UnreadCount))
		}

		line1 := fmt.Sprintf("  %s %s %s%s", styledIcon, title, branchStr, unread)
		line2 := fmt.Sprintf("    %s · %s", agentStr, statusStr)

		if i == m.Cursor {
			line1 = t.SelectedItem.Width(m.Width - 2).Render(line1)
			line2 = t.SelectedItem.Width(m.Width - 2).Render(line2)
		}

		b.WriteString(line1)
		b.WriteString("\n")
		b.WriteString(line2)
		b.WriteString("\n")

		if i < len(m.Workspaces)-1 {
			b.WriteString("\n")
		}
	}

	content := b.String()
	style := t.SidebarBorder.Width(m.Width).Height(m.Height)
	return style.Render(content)
}

func (m Model) ShortHelp() string {
	t := theme.DefaultTheme()
	help := []string{
		t.HelpKey.Render("j/k") + t.HelpDesc.Render(" navigate"),
		t.HelpKey.Render("enter") + t.HelpDesc.Render(" attach"),
		t.HelpKey.Render("n") + t.HelpDesc.Render(" new"),
		t.HelpKey.Render("?") + t.HelpDesc.Render(" help"),
		t.HelpKey.Render("q") + t.HelpDesc.Render(" quit"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(help, "  "))
}
