package tmux

import (
	"regexp"
	"strings"
)

var DefaultRunningPatterns = []string{
	`esc to interrupt`,            // Claude Code status bar during processing
	`[◐◑◒◓⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`,        // Legacy spinner characters
}

var DefaultDonePatterns = []string{
	`\? for shortcuts`,            // Claude Code status bar at prompt
	`⏵⏵`,                         // Claude Code accept edits mode
	`^> $`,                        // Legacy Claude prompt
	`✓ Task completed`,
}

var DefaultErrorPatterns = []string{
	`Error:`,
	`Failed to`,
	`✗`,
}

var defaultIdlePatterns = []string{
	`[\$%#]\s*$`,                  // Shell prompt (❯ removed — used by Claude Code)
}

// StatusDetector detects pane status from capture-pane output.
type StatusDetector struct {
	runningRegexps []*regexp.Regexp
	doneRegexps    []*regexp.Regexp
	errorRegexps   []*regexp.Regexp
	idleRegexps    []*regexp.Regexp
}

// NewStatusDetector creates a detector with default patterns.
func NewStatusDetector() *StatusDetector {
	return &StatusDetector{
		runningRegexps: compilePatterns(DefaultRunningPatterns),
		doneRegexps:    compilePatterns(DefaultDonePatterns),
		errorRegexps:   compilePatterns(DefaultErrorPatterns),
		idleRegexps:    compilePatterns(defaultIdlePatterns),
	}
}

// NewStatusDetectorWithPatterns creates a detector with custom patterns.
// Nil slices fall back to defaults.
func NewStatusDetectorWithPatterns(running, done, errorPat []string) *StatusDetector {
	d := NewStatusDetector()
	if len(running) > 0 {
		d.runningRegexps = compilePatterns(running)
	}
	if len(done) > 0 {
		d.doneRegexps = compilePatterns(done)
	}
	if len(errorPat) > 0 {
		d.errorRegexps = compilePatterns(errorPat)
	}
	return d
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var result []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			result = append(result, re)
		}
	}
	return result
}

// Detect determines the pane status from capture-pane output lines.
func (d *StatusDetector) Detect(lines []string) PaneStatus {
	var lastLines []string
	for i := len(lines) - 1; i >= 0 && len(lastLines) < 5; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			lastLines = append(lastLines, lines[i])
		}
	}

	if len(lastLines) == 0 {
		return StatusUnknown
	}

	// Check running first (Claude Code status bar is most reliable)
	for _, line := range lastLines {
		for _, re := range d.runningRegexps {
			if re.MatchString(line) {
				return StatusRunning
			}
		}
	}

	// Check done patterns
	for _, line := range lastLines {
		for _, re := range d.doneRegexps {
			if re.MatchString(line) {
				return StatusDone
			}
		}
	}

	// Check error patterns
	for _, line := range lastLines {
		for _, re := range d.errorRegexps {
			if re.MatchString(line) {
				return StatusError
			}
		}
	}

	// Check idle (shell prompt) — only last line
	for _, line := range lastLines[:1] {
		for _, re := range d.idleRegexps {
			if re.MatchString(line) {
				return StatusIdle
			}
		}
	}

	return StatusUnknown
}
