package agent

import (
	"regexp"
	"time"
)

type AgentType string

const (
	Claude   AgentType = "claude"
	Codex    AgentType = "codex"
	Gemini   AgentType = "gemini"
	OpenCode AgentType = "opencode"
	Aider    AgentType = "aider"
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

type AgentDef struct {
	Name            AgentType
	Binary          string
	LaunchFlags     []string
	YoloFlags       []string
	InputDelay      time.Duration
	IgnorePatterns  []*regexp.Regexp
	WorkingPatterns []*regexp.Regexp
	WaitingPatterns []*regexp.Regexp
	IdlePatterns    []*regexp.Regexp
	ErrorPatterns   []*regexp.Regexp
}
