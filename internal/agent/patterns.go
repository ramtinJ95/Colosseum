package agent

import "regexp"

var (
	BrailleSpinner   = regexp.MustCompile(`[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	CommonPromptChars = regexp.MustCompile(`^\s*[>$❯]\s*$`)
	RateLimitPattern = regexp.MustCompile(`(?i)(rate.?limit|429|too many requests)`)
	PanicPattern     = regexp.MustCompile(`(?i)(panic:|fatal error:|segmentation fault)`)
	AuthErrorPattern = regexp.MustCompile(`(?i)(unauthorized|authentication failed|invalid.*api.*key|EAUTH)`)
)
