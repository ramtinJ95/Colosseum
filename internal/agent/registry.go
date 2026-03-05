package agent

import "sort"

var registry = make(map[AgentType]*AgentDef)

var supportedAgents = []AgentType{Claude, Codex}

func Register(def *AgentDef) {
	registry[def.Name] = def
}

func Get(name AgentType) (*AgentDef, bool) {
	def, ok := registry[name]
	return def, ok
}

func Available() []AgentType {
	types := make([]AgentType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func Supported() []AgentType {
	types := make([]AgentType, len(supportedAgents))
	copy(types, supportedAgents)
	return types
}

func IsSupported(name AgentType) bool {
	for _, supported := range supportedAgents {
		if name == supported {
			return true
		}
	}
	return false
}
