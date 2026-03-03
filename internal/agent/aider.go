package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Aider,
		Binary:      "aider",
		LaunchFlags: []string{},
		YoloFlags:   []string{"--yes-always"},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
			regexp.MustCompile(`(?i)(thinking|processing|applying)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(y/n|yes/no)`),
			regexp.MustCompile(`\?\s*$`),
			regexp.MustCompile(`(?i)(allow|confirm)`),
		},
		IdlePatterns: []*regexp.Regexp{
			CommonPromptChars,
			regexp.MustCompile(`^>\s*$`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
