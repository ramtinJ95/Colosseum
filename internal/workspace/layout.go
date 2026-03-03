package workspace

type LayoutType string

const (
	LayoutAgent         LayoutType = "agent"
	LayoutAgentShell    LayoutType = "agent-shell"
	LayoutAgentShellLogs LayoutType = "agent-shell-logs"
)

func ValidLayouts() []LayoutType {
	return []LayoutType{LayoutAgent, LayoutAgentShell, LayoutAgentShellLogs}
}

func (l LayoutType) PaneCount() int {
	switch l {
	case LayoutAgent:
		return 1
	case LayoutAgentShell:
		return 2
	case LayoutAgentShellLogs:
		return 3
	default:
		return 1
	}
}
