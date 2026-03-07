package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/config"
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
	return ThemeFromConfig(config.Default().Theme)
}

func ThemeFromConfig(tc config.ThemeConfig) Theme {
	return Theme{
		AppTitle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tc.AppTitle)),
		SidebarBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(tc.Border)),
		PreviewBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(tc.Border)),
		SelectedItem:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tc.SelectedFG)).Background(lipgloss.Color(tc.SelectedBG)),
		NormalItem:    lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Normal)),
		StatusWorking: lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Working)),
		StatusWaiting: lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Waiting)),
		StatusIdle:    lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Idle)),
		StatusStopped: lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Stopped)),
		StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Error)),
		StatusUnknown: lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Idle)),
		BranchName:    lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Branch)),
		AgentName:     lipgloss.NewStyle().Foreground(lipgloss.Color(tc.AgentName)),
		UnreadBadge:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tc.Waiting)),
		PreviewTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tc.AppTitle)).Padding(0, 1),
		HelpKey:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(tc.HelpKey)),
		HelpDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color(tc.HelpDesc)),
		Dim:           lipgloss.NewStyle().Foreground(lipgloss.Color(tc.Dim)),
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
