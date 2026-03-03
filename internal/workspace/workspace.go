package workspace

import (
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
)

type Workspace struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	AgentType   agent.AgentType   `json:"agent_type"`
	ProjectPath string            `json:"project_path"`
	Branch      string            `json:"branch"`
	Layout      LayoutType        `json:"layout"`
	Status      agent.Status      `json:"status"`
	SessionName string            `json:"session_name"`
	PaneTargets map[string]string `json:"pane_targets"`
	UnreadCount int               `json:"unread_count"`
	CreatedAt   time.Time         `json:"created_at"`
}
