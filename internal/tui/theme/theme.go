package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/agent"
)

type Theme struct {
	AppTitle       lipgloss.Style
	SidebarBorder  lipgloss.Style
	PreviewBorder  lipgloss.Style
	SelectedItem   lipgloss.Style
	NormalItem     lipgloss.Style
	StatusWorking  lipgloss.Style
	StatusWaiting  lipgloss.Style
	StatusIdle     lipgloss.Style
	StatusStopped  lipgloss.Style
	StatusError    lipgloss.Style
	StatusUnknown  lipgloss.Style
	BranchName     lipgloss.Style
	AgentName      lipgloss.Style
	UnreadBadge    lipgloss.Style
	PreviewTitle   lipgloss.Style
	HelpKey        lipgloss.Style
	HelpDesc       lipgloss.Style
	Dim            lipgloss.Style
}

func DefaultTheme() Theme {
	return Theme{
		AppTitle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
		SidebarBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		PreviewBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
		SelectedItem:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Background(lipgloss.Color("236")),
		NormalItem:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		StatusWorking: lipgloss.NewStyle().Foreground(lipgloss.Color("82")),
		StatusWaiting: lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		StatusIdle:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		StatusStopped: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		BranchName:    lipgloss.NewStyle().Foreground(lipgloss.Color("109")),
		AgentName:     lipgloss.NewStyle().Foreground(lipgloss.Color("140")),
		UnreadBadge:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")),
		PreviewTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).Padding(0, 1),
		HelpKey:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")),
		HelpDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

func (t Theme) StatusStyle(s agent.Status) lipgloss.Style {
	switch s {
	case agent.StatusWorking:
		return t.StatusWorking
	case agent.StatusWaiting:
		return t.StatusWaiting
	case agent.StatusIdle:
		return t.StatusIdle
	case agent.StatusStopped:
		return t.StatusStopped
	case agent.StatusError:
		return t.StatusError
	default:
		return t.StatusUnknown
	}
}

func StatusIcon(s agent.Status) string {
	switch s {
	case agent.StatusWorking:
		return "●"
	case agent.StatusWaiting:
		return "◉"
	case agent.StatusIdle:
		return "○"
	case agent.StatusStopped:
		return "■"
	case agent.StatusError:
		return "✗"
	default:
		return "?"
	}
}
