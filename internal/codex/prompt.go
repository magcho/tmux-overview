package codex

import (
	"regexp"
	"strings"
)

var (
	permissionPromptPattern = regexp.MustCompile(`(?i)allow .* to run tool|field \d+/\d+|enter to submit \| esc to cancel`)
	commandApprovalPattern  = regexp.MustCompile(`(?i)would you like to run the following command\?|press enter to confirm or esc to cancel`)
	commandChoicePattern    = regexp.MustCompile(`(?i)yes, proceed|don't ask again|tell codex what to do differently`)
	hookStatusPattern       = regexp.MustCompile(`(?i)^(?:[•*-]\s*)?(?:running [[:alnum:]_:-]+ hook|[[:alnum:]_:-]+ hook \(completed\))$`)
)

// DetectWaiting reports whether the visible Codex terminal output indicates
// that human confirmation is required, plus a short summary for state/notify use.
func DetectWaiting(preview []string) (bool, string) {
	lines := trailingNonEmptyLines(preview, 16)
	if len(lines) == 0 {
		return false, ""
	}

	window := strings.Join(lines, "\n")
	switch {
	case permissionPromptPattern.MatchString(window):
		return true, summarizeWaitingPrompt(lines)
	case commandApprovalPattern.MatchString(window) && commandChoicePattern.MatchString(window):
		return true, summarizeWaitingPrompt(lines)
	case hookStatusPattern.MatchString(lines[len(lines)-1]):
		return true, summarizeWaitingPrompt(lines)
	default:
		return false, ""
	}
}

func summarizeWaitingPrompt(lines []string) string {
	for _, line := range lines {
		line = strings.TrimSpace(stripANSI(line))
		if strings.HasPrefix(line, "$ ") {
			return line
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(stripANSI(line))
		if strings.HasPrefix(line, "Reason:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(stripANSI(line))
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "would you like to run the following command?"):
			return line
		case strings.Contains(lower, "allow ") && strings.Contains(lower, " to run tool"):
			return line
		}
	}

	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(stripANSI(lines[i]))
		if line != "" {
			return line
		}
	}
	return ""
}

func trailingNonEmptyLines(lines []string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	var trimmed []string
	for i := len(lines) - 1; i >= 0 && len(trimmed) < limit; i-- {
		line := strings.TrimSpace(stripANSI(lines[i]))
		if line == "" {
			continue
		}
		trimmed = append(trimmed, line)
	}

	for i, j := 0, len(trimmed)-1; i < j; i, j = i+1, j-1 {
		trimmed[i], trimmed[j] = trimmed[j], trimmed[i]
	}
	return trimmed
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEscape {
			if ch >= '@' && ch <= '~' {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}

	return b.String()
}
