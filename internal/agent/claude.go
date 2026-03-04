package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Claude,
		Binary:      "claude",
		LaunchFlags: []string{},
		YoloFlags:   []string{"--dangerously-skip-permissions"},
		WorkingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\(esc to interrupt\)`),
			BrailleSpinner,
		},
		WaitingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(yes,?\s*allow|allow\s*(once|always))`),
			regexp.MustCompile(`(?i)(do you want|would you like|should i|shall i)`),
			regexp.MustCompile(`\?\s*$`),
			regexp.MustCompile(`(?i)permission`),
		},
		IdlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^>\s*$`),
			regexp.MustCompile(`^\$\s*$`),
			regexp.MustCompile(`^❯\s*$`),
			regexp.MustCompile(`-- INSERT --`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
