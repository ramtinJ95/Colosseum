package status

import "github.com/ramtinj/colosseum/internal/agent"

type Update struct {
	WorkspaceID string
	Previous    agent.Status
	Current     agent.Status
	PaneContent string
}
