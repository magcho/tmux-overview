package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/magcho/tmux-overview/internal/config"
)

func TestDetectTerminalApp(t *testing.T) {
	tests := []struct {
		name     string
		cfgApp   string
		envVal   string
		expected string
	}{
		{"config overrides env", "WezTerm", "ghostty", "WezTerm"},
		{"ghostty", "", "ghostty", "Ghostty"},
		{"iterm", "", "iTerm.app", "iTerm2"},
		{"apple terminal", "", "Apple_Terminal", "Terminal"},
		{"wezterm", "", "WezTerm", "WezTerm"},
		{"alacritty", "", "alacritty", "Alacritty"},
		{"unknown", "", "SomeTerminal", "SomeTerminal"},
		{"empty falls back to Terminal", "", "", "Terminal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", tt.envVal)
			cfg := config.NotifyConfig{TerminalApp: tt.cfgApp}
			got := detectTerminalApp(cfg)
			if got != tt.expected {
				t.Errorf("detectTerminalApp() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"long string truncated", "hello world", 5, "hello..."},
		{"multibyte string", "日本語テスト", 3, "日本語..."},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateBody(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateBody(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestReadLastLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Write a file with 10 lines
	var content string
	for i := 0; i < 10; i++ {
		content += `{"line":` + string(rune('0'+i)) + "}\n"
	}
	os.WriteFile(path, []byte(content), 0600)

	t.Run("fewer lines than n", func(t *testing.T) {
		lines, err := readLastLines(path, 20)
		if err != nil {
			t.Fatalf("readLastLines: %v", err)
		}
		if len(lines) != 10 {
			t.Errorf("got %d lines, want 10", len(lines))
		}
	})

	t.Run("exactly n lines", func(t *testing.T) {
		lines, err := readLastLines(path, 10)
		if err != nil {
			t.Fatalf("readLastLines: %v", err)
		}
		if len(lines) != 10 {
			t.Errorf("got %d lines, want 10", len(lines))
		}
	})

	t.Run("more lines than n", func(t *testing.T) {
		lines, err := readLastLines(path, 3)
		if err != nil {
			t.Fatalf("readLastLines: %v", err)
		}
		if len(lines) != 3 {
			t.Errorf("got %d lines, want 3", len(lines))
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := readLastLines(filepath.Join(dir, "nonexistent"), 5)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestExtractNotificationContext(t *testing.T) {
	dir := t.TempDir()

	t.Run("empty path returns empty", func(t *testing.T) {
		got := extractNotificationContext("")
		if got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})

	t.Run("extracts tool_use with description", func(t *testing.T) {
		path := filepath.Join(dir, "notify.jsonl")
		content := `{"type":"assistant","message":{"content":[{"type":"text","text":"Let me check the status."},{"type":"tool_use","name":"Bash","input":{"command":"git status","description":"Check git status"}}]}}
`
		os.WriteFile(path, []byte(content), 0600)

		got := extractNotificationContext(path)
		if got == "" {
			t.Error("expected non-empty context")
		}
		if !contains(got, "[Bash]") {
			t.Errorf("expected [Bash] in context, got %q", got)
		}
		if !contains(got, "Check git status") {
			t.Errorf("expected description in context, got %q", got)
		}
	})

	t.Run("extracts tool_use with command fallback", func(t *testing.T) {
		path := filepath.Join(dir, "notify2.jsonl")
		content := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/tmp/test.go"}}]}}
`
		os.WriteFile(path, []byte(content), 0600)

		got := extractNotificationContext(path)
		if !contains(got, "[Read]") || !contains(got, "/tmp/test.go") {
			t.Errorf("expected [Read] /tmp/test.go, got %q", got)
		}
	})
}

func TestExtractStopSummary(t *testing.T) {
	dir := t.TempDir()

	t.Run("empty path returns default", func(t *testing.T) {
		got := extractStopSummary("")
		if got != "作業が完了しました" {
			t.Errorf("expected default message, got %q", got)
		}
	})

	t.Run("extracts last text message", func(t *testing.T) {
		path := filepath.Join(dir, "stop.jsonl")
		content := `{"type":"assistant","message":{"content":[{"type":"text","text":"First message"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"Task completed successfully. All tests pass."}]}}
`
		os.WriteFile(path, []byte(content), 0600)

		got := extractStopSummary(path)
		if !contains(got, "Task completed successfully") {
			t.Errorf("expected last message, got %q", got)
		}
	})

	t.Run("no text entries returns default", func(t *testing.T) {
		path := filepath.Join(dir, "stop2.jsonl")
		content := `{"type":"user","message":{"content":[{"type":"text","text":"hello"}]}}
`
		os.WriteFile(path, []byte(content), 0600)

		got := extractStopSummary(path)
		if got != "作業が完了しました" {
			t.Errorf("expected default, got %q", got)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
