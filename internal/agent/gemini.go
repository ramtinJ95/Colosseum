package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Gemini,
		Binary:      "gemini",
		LaunchFlags: []string{},
		YoloFlags:   []string{"-y"},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
			regexp.MustCompile(`(?i)(thinking|working|processing)`),
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(allow|deny|approve)`),
			regexp.MustCompile(`\?\s*$`),
		},
		IdlePatterns: []*regexp.Regexp{
			CommonPromptChars,
			regexp.MustCompile(`^❯\s*$`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
