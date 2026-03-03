package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Codex,
		Binary:      "codex",
		LaunchFlags: []string{},
		YoloFlags:   []string{"--approval-mode", "full-auto"},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
			regexp.MustCompile(`(?i)(thinking|generating|processing)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(approve|deny|explain)`),
			regexp.MustCompile(`(?i)(yes|no|edit)\s*$`),
		},
		IdlePatterns: []*regexp.Regexp{
			CommonPromptChars,
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
		},
	})
}
