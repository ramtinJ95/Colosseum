package workspace

type LayoutType string

const (
	LayoutAgent          LayoutType = "agent"
	LayoutAgentShell     LayoutType = "agent-shell"
	LayoutAgentShellLogs LayoutType = "agent-shell-logs"
)

var validLayouts = []LayoutType{LayoutAgent, LayoutAgentShell, LayoutAgentShellLogs}

func ValidLayouts() []LayoutType {
	layouts := make([]LayoutType, len(validLayouts))
	copy(layouts, validLayouts)
	return layouts
}

func IsValidLayout(layout LayoutType) bool {
	for _, candidate := range validLayouts {
		if layout == candidate {
			return true
		}
	}
	return false
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
