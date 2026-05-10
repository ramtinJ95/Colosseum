package workspace

import (
	"strings"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
)

func AgentSendOptions(ws Workspace, text string) tmux.SendOptions {
	opts := tmux.SendOptions{}
	if def, ok := agent.Get(ws.AgentType); ok {
		opts.InputDelay = def.InputDelay
		opts.ForcePaste = def.PasteSingleLine && !strings.Contains(text, "\n")
		opts.DisableBracketedPaste = def.DisableBracketedPasteForMultiline && strings.Contains(text, "\n")
	}
	return opts
}
