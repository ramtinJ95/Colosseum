package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        Claude,
		Binary:      "claude",
		LaunchFlags: []string{},
		YoloFlags:   []string{"--dangerously-skip-permissions"},
		IgnorePatterns: []*regexp.Regexp{
			regexp.MustCompile(`Tokens:.*Remaining:`),
			regexp.MustCompile(`^\s*Opus .* \| .*`),
			regexp.MustCompile(`^[\s─▪]+$`),
			regexp.MustCompile(`^\s*--\s+(INSERT|NORMAL)\s+--`),
		},
		WorkingPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\(esc to interrupt\)`),
			BrailleSpinner,
			regexp.MustCompile(`(?m)^\s*✻\s+Cooked for\b`),
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
			regexp.MustCompile(`^❯`),
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
