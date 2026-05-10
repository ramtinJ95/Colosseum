package agent

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type AgentType string

const (
	Claude   AgentType = "claude"
	Codex    AgentType = "codex"
	Gemini   AgentType = "gemini"
	OpenCode AgentType = "opencode"
	Aider    AgentType = "aider"
	PiAgent  AgentType = "pi-agent"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusWorking
	StatusWaiting
	StatusIdle
	StatusStopped
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusWorking:
		return "Working"
	case StatusWaiting:
		return "Waiting"
	case StatusIdle:
		return "Idle"
	case StatusStopped:
		return "Stopped"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

func ParseStatus(value string) (Status, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "unknown":
		return StatusUnknown, nil
	case "working":
		return StatusWorking, nil
	case "waiting", "blocked":
		return StatusWaiting, nil
	case "idle":
		return StatusIdle, nil
	case "stopped":
		return StatusStopped, nil
	case "error":
		return StatusError, nil
	default:
		return StatusUnknown, fmt.Errorf("unknown status %q", value)
	}
}

type AgentDef struct {
	Name                              AgentType
	Binary                            string
	LaunchFlags                       []string
	YoloFlags                         []string
	InputDelay                        time.Duration
	PasteSingleLine                   bool
	DisableBracketedPasteForMultiline bool
	IgnorePatterns                    []*regexp.Regexp
	WorkingPatterns                   []*regexp.Regexp
	WaitingPatterns                   []*regexp.Regexp
	IdlePatterns                      []*regexp.Regexp
	ErrorPatterns                     []*regexp.Regexp
}
