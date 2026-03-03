package agent

import "sort"

var registry = make(map[AgentType]*AgentDef)

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
