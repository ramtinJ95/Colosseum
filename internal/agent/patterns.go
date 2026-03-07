package agent

import "regexp"

var (
	BrailleSpinner      = regexp.MustCompile(`[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	CommonPromptChars   = regexp.MustCompile(`^\s*[>$❯]\s*$`)
	ChoiceMenuPattern   = regexp.MustCompile(`(?m)^\s*❯\s*\d+\.`)
	ChoicePromptPattern = regexp.MustCompile(`(?i)(which .* (prefer|use)|which approach|which testing framework|what .* want to tackle|how would you like|select (one|an option)|choose (one|an option))`)
	RateLimitPattern    = regexp.MustCompile(`(?i)(rate.?limit|429|too many requests)`)
	PanicPattern        = regexp.MustCompile(`(?i)(panic:|fatal error:|segmentation fault)`)
	AuthErrorPattern    = regexp.MustCompile(`(?i)(unauthorized|authentication failed|invalid.*api.*key|EAUTH)`)
)
