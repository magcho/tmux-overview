package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magcho/tmux-overview/internal/config"
)

// sendNotification sends a macOS notification via terminal-notifier.
// It spawns terminal-notifier as a detached process and returns immediately.
func sendNotification(title, body string, paneInfo tmuxPaneInfo, cfg config.NotifyConfig) {
	notifierPath, err := exec.LookPath("terminal-notifier")
	if err != nil {
		return // terminal-notifier not installed; skip silently
	}

	tovPath, err := os.Executable()
	if err != nil {
		return
	}

	// Build focus command for notification click
	target := fmt.Sprintf("%s:%d.%d", paneInfo.SessionName, paneInfo.WindowIndex, paneInfo.PaneIndex)
	app := detectTerminalApp(cfg)

	executeCmd := fmt.Sprintf("%s focus --socket '%s' --target '%s'",
		tovPath, paneInfo.SocketPath, target)
	if app != "" {
		executeCmd += fmt.Sprintf(" --app '%s'", app)
	}

	args := []string{
		"-title", title,
		"-message", body,
		"-group", "tov-" + paneInfo.PaneID,
		"-execute", executeCmd,
	}

	if cfg.Sound != "" {
		args = append(args, "-sound", cfg.Sound)
	}
	if cfg.Icon != "" {
		args = append(args, "-appIcon", cfg.Icon)
	}

	cmd := exec.Command(notifierPath, args...)
	// Start without Wait — the process runs independently after tov hook exits
	cmd.Start()
}

// detectTerminalApp determines the terminal application name.
func detectTerminalApp(cfg config.NotifyConfig) string {
	if cfg.TerminalApp != "" {
		return cfg.TerminalApp
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	switch strings.ToLower(termProgram) {
	case "ghostty":
		return "Ghostty"
	case "iterm.app":
		return "iTerm2"
	case "apple_terminal":
		return "Terminal"
	case "wezterm":
		return "WezTerm"
	case "alacritty":
		return "Alacritty"
	case "":
		return "Terminal"
	default:
		return termProgram
	}
}

// Transcript parsing types

type transcriptEntry struct {
	Type    string `json:"type"`
	Message struct {
		Content []transcriptContent `json:"content"`
	} `json:"message"`
}

type transcriptContent struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type toolInput struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
}

// extractNotificationContext parses the transcript to extract the last tool_use context.
func extractNotificationContext(transcriptPath string) string {
	if transcriptPath == "" {
		return ""
	}

	lines, err := readLastLines(transcriptPath, 20)
	if err != nil {
		return ""
	}

	// Search backwards for tool_use and preceding text context
	var lastToolUse string
	var lastContext string

	for i := len(lines) - 1; i >= 0; i-- {
		var entry transcriptEntry
		if err := json.Unmarshal([]byte(lines[i]), &entry); err != nil {
			continue
		}
		if entry.Type != "assistant" {
			continue
		}

		for _, c := range entry.Message.Content {
			switch c.Type {
			case "tool_use":
				if lastToolUse == "" {
					lastToolUse = formatToolContext(c)
				}
			case "text":
				if lastContext == "" && c.Text != "" {
					lastContext = truncateBody(c.Text, 100)
				}
			}
		}

		if lastToolUse != "" {
			break
		}
	}

	if lastContext != "" && lastToolUse != "" {
		return lastContext + "\n" + lastToolUse
	}
	if lastToolUse != "" {
		return lastToolUse
	}
	return lastContext
}

// extractStopSummary parses the transcript to find the last assistant text message.
func extractStopSummary(transcriptPath string) string {
	if transcriptPath == "" {
		return "作業が完了しました"
	}

	lines, err := readLastLines(transcriptPath, 20)
	if err != nil {
		return "作業が完了しました"
	}

	// Search backwards for the last assistant text
	for i := len(lines) - 1; i >= 0; i-- {
		var entry transcriptEntry
		if err := json.Unmarshal([]byte(lines[i]), &entry); err != nil {
			continue
		}
		if entry.Type != "assistant" {
			continue
		}

		// Find the last text content in this entry
		var lastText string
		for _, c := range entry.Message.Content {
			if c.Type == "text" && c.Text != "" {
				lastText = c.Text
			}
		}
		if lastText != "" {
			return truncateBody(lastText, 200)
		}
	}

	return "作業が完了しました"
}

// formatToolContext formats a tool_use entry for notification display.
func formatToolContext(c transcriptContent) string {
	var input toolInput
	if c.Input != nil {
		json.Unmarshal(c.Input, &input)
	}

	detail := input.Description
	if detail == "" {
		detail = input.Command
	}
	if detail == "" {
		detail = input.FilePath
	}
	if detail == "" {
		detail = input.Pattern
	}

	if detail != "" {
		return fmt.Sprintf("[%s] %s", c.Name, truncateBody(detail, 150))
	}
	return fmt.Sprintf("[%s]", c.Name)
}

// readLastLines reads the last n lines of a file efficiently.
func readLastLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// For efficiency with large files, seek near the end
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// If file is small enough, just read all lines
	const chunkSize = 8192
	if info.Size() <= chunkSize {
		var lines []string
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, chunkSize), chunkSize*2)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if len(lines) > n {
			return lines[len(lines)-n:], nil
		}
		return lines, nil
	}

	// Read from end in chunks to find last n lines
	offset := info.Size() - chunkSize
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, chunkSize), chunkSize*2)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// If we seeked, the first line may be partial — skip it
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	if len(lines) > n {
		return lines[len(lines)-n:], nil
	}
	return lines, nil
}

// truncateBody truncates a string to maxLen runes.
func truncateBody(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
