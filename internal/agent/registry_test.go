package agent

import (
	"regexp"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	testType := AgentType("test-agent")

	Register(&AgentDef{
		Name:   testType,
		Binary: "test-bin",
		WorkingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`working`),
		},
	})

	def, ok := Get(testType)
	if !ok {
		t.Fatal("expected to find registered agent")
	}
	if def.Binary != "test-bin" {
		t.Errorf("expected binary %q, got %q", "test-bin", def.Binary)
	}
	if len(def.WorkingPatterns) != 1 {
		t.Errorf("expected 1 working pattern, got %d", len(def.WorkingPatterns))
	}

	_, ok = Get(AgentType("nonexistent"))
	if ok {
		t.Error("expected not to find nonexistent agent")
	}
}

func TestAvailable(t *testing.T) {
	types := Available()
	if len(types) < 6 {
		t.Fatalf("expected at least 6 agent types, got %d", len(types))
	}

	for i := 1; i < len(types); i++ {
		if types[i] < types[i-1] {
			t.Errorf("agent types not sorted: %q came after %q", types[i], types[i-1])
		}
	}
}

func TestBuiltinAgentsRegistered(t *testing.T) {
	builtins := []AgentType{Claude, Codex, Gemini, OpenCode, Aider, PiAgent}

	for _, agentType := range builtins {
		def, ok := Get(agentType)
		if !ok {
			t.Errorf("expected builtin agent %q to be registered", agentType)
			continue
		}
		if def.Binary == "" {
			t.Errorf("agent %q has empty binary", agentType)
		}
		if len(def.WorkingPatterns) == 0 {
			t.Errorf("agent %q has no working patterns", agentType)
		}
		if len(def.ErrorPatterns) == 0 {
			t.Errorf("agent %q has no error patterns", agentType)
		}
	}
}

func TestSupported(t *testing.T) {
	if got := Supported(); len(got) != 4 {
		t.Fatalf("Supported() len = %d, want 4", len(got))
	}

	if !IsSupported(Claude) {
		t.Fatal("expected claude to be supported")
	}
	if !IsSupported(Codex) {
		t.Fatal("expected codex to be supported")
	}
	if !IsSupported(OpenCode) {
		t.Fatal("expected opencode to be supported")
	}
	if !IsSupported(PiAgent) {
		t.Fatal("expected pi-agent to be supported")
	}
	if IsSupported(Gemini) {
		t.Fatal("expected gemini to be unsupported")
	}
}

func TestPiAgentLaunchDefinition(t *testing.T) {
	def, ok := Get(PiAgent)
	if !ok {
		t.Fatal("expected pi-agent to be registered")
	}
	if def.Binary != "pi" {
		t.Fatalf("binary = %q, want %q", def.Binary, "pi")
	}
	if len(def.LaunchFlags) != 0 {
		t.Fatalf("launch flags = %v, want none", def.LaunchFlags)
	}
}
