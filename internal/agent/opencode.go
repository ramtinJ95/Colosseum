package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        OpenCode,
		Binary:      "opencode",
		LaunchFlags: []string{},
		YoloFlags:   []string{},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
			regexp.MustCompile(`(?i)(running|processing)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(confirm|approve)`),
			regexp.MustCompile(`\?\s*$`),
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
