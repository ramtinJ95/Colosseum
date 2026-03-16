package agent

import "regexp"

func init() {
	Register(&AgentDef{
		Name:        PiAgent,
		Binary:      "pi",
		LaunchFlags: []string{},
		YoloFlags:   []string{},
		IgnorePatterns: []*regexp.Regexp{
			regexp.MustCompile(`^[\s─]+$`),
			regexp.MustCompile(`^\s*(~?/|/).+\([^\n)]+\)\s*$`),
		},
		WorkingPatterns: []*regexp.Regexp{
			BrailleSpinner,
		},
		WaitingPatterns: nil,
		IdlePatterns: []*regexp.Regexp{
			regexp.MustCompile(`--\s+(INSERT|NORMAL)\s+--`),
			regexp.MustCompile(`^\s*(?:↑\S+\s+↓\S+\s+)?\$\d+(?:\.\d+)?(?:\s+\(sub\))?\s+\d+(?:\.\d+)?/[0-9.]+[KMG]?\s+\d+(?:\.\d+)?%\s+.+$`),
			CommonPromptChars,
		},
		ErrorPatterns: []*regexp.Regexp{
			RateLimitPattern,
			PanicPattern,
			AuthErrorPattern,
		},
	})
}
