package hook

import (
	"testing"

	"github.com/magcho/tmux-overview/internal/state"
)

func TestDetectAgentDefault(t *testing.T) {
	// Without Codex-specific env vars, should default to Claude.
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SANDBOX", "")
	t.Setenv("CODEX_SANDBOX_NETWORK_DISABLED", "")
	t.Setenv("CODEX_CI", "")
	if got := DetectAgent(); got != AgentClaude {
		t.Errorf("DetectAgent() = %q, want %q", got, AgentClaude)
	}
}

func TestDetectAgentCodex(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "thread-123")
	if got := DetectAgent(); got != AgentCodex {
		t.Errorf("DetectAgent() = %q, want %q", got, AgentCodex)
	}
}

func TestDetectAgentCodexByFallbackEnv(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SANDBOX", "seatbelt")
	t.Setenv("CODEX_SANDBOX_NETWORK_DISABLED", "")
	t.Setenv("CODEX_CI", "")
	if got := DetectAgent(); got != AgentCodex {
		t.Errorf("DetectAgent() = %q, want %q", got, AgentCodex)
	}
}

func TestGetAgentClaude(t *testing.T) {
	def, ok := GetAgent(AgentClaude)
	if !ok {
		t.Fatal("GetAgent(AgentClaude) not found")
	}
	if def.Name != AgentClaude {
		t.Errorf("Name = %q, want %q", def.Name, AgentClaude)
	}
	if def.DisplayLabel != "Claude Code" {
		t.Errorf("DisplayLabel = %q, want %q", def.DisplayLabel, "Claude Code")
	}
	if len(def.HookEvents) != 6 {
		t.Errorf("HookEvents length = %d, want 6", len(def.HookEvents))
	}
}

func TestGetAgentCodex(t *testing.T) {
	def, ok := GetAgent(AgentCodex)
	if !ok {
		t.Fatal("GetAgent(AgentCodex) not found")
	}
	if def.Name != AgentCodex {
		t.Errorf("Name = %q, want %q", def.Name, AgentCodex)
	}
	if def.DisplayLabel != "Codex" {
		t.Errorf("DisplayLabel = %q, want %q", def.DisplayLabel, "Codex")
	}
	if len(def.HookEvents) != 4 {
		t.Errorf("HookEvents length = %d, want 4", len(def.HookEvents))
	}
}

func TestGetAgentUnknown(t *testing.T) {
	_, ok := GetAgent("unknown")
	if ok {
		t.Error("GetAgent(unknown) should return false")
	}
}

func TestAllAgents(t *testing.T) {
	all := AllAgents()
	if len(all) < 2 {
		t.Errorf("AllAgents() returned %d agents, want at least 2", len(all))
	}

	names := make(map[AgentName]bool)
	for _, a := range all {
		names[a.Name] = true
	}
	if !names[AgentClaude] {
		t.Error("AllAgents() missing claude")
	}
	if !names[AgentCodex] {
		t.Error("AllAgents() missing codex")
	}
}

func TestDefaultStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		current  state.Status
		expected state.Status
	}{
		{"SessionStart sets registered", "SessionStart", "", state.StatusRegistered},
		{"UserPromptSubmit sets running", "UserPromptSubmit", state.StatusRegistered, state.StatusRunning},
		{"PreToolUse sets running", "PreToolUse", state.StatusWaiting, state.StatusRunning},
		{"Stop sets done", "Stop", state.StatusRunning, state.StatusDone},
		{"Unknown preserves current", "PostToolUse", state.StatusRunning, state.StatusRunning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultStatusTransition(tt.event, hookInput{}, tt.current)
			if got != tt.expected {
				t.Errorf("defaultStatusTransition(%q, ..., %q) = %q, want %q",
					tt.event, tt.current, got, tt.expected)
			}
		})
	}
}

func TestClaudeStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		input    hookInput
		current  state.Status
		expected state.Status
	}{
		{
			"Notification permission_prompt sets waiting",
			"Notification",
			hookInput{NotificationType: "permission_prompt"},
			state.StatusRunning,
			state.StatusWaiting,
		},
		{
			"Notification idle_prompt sets done",
			"Notification",
			hookInput{NotificationType: "idle_prompt"},
			state.StatusRunning,
			state.StatusDone,
		},
		{
			"Notification elicitation_dialog sets waiting",
			"Notification",
			hookInput{NotificationType: "elicitation_dialog"},
			state.StatusRunning,
			state.StatusWaiting,
		},
		{
			"Non-Notification falls through to default",
			"UserPromptSubmit",
			hookInput{},
			state.StatusRegistered,
			state.StatusRunning,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := claudeStatusTransition(tt.event, tt.input, tt.current)
			if got != tt.expected {
				t.Errorf("claudeStatusTransition(%q, ..., %q) = %q, want %q",
					tt.event, tt.current, got, tt.expected)
			}
		})
	}
}

func TestResolveAgentFallback(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SANDBOX", "")
	t.Setenv("CODEX_SANDBOX_NETWORK_DISABLED", "")
	t.Setenv("CODEX_CI", "")
	def := resolveAgent()
	if def.Name != AgentClaude {
		t.Errorf("resolveAgent() = %q, want %q", def.Name, AgentClaude)
	}
}

func TestResolveAgentCodex(t *testing.T) {
	t.Setenv("CODEX_THREAD_ID", "thread-abc")
	def := resolveAgent()
	if def.Name != AgentCodex {
		t.Errorf("resolveAgent() = %q, want %q", def.Name, AgentCodex)
	}
}

func TestClaudeShouldRemoveOnEvent(t *testing.T) {
	def, _ := GetAgent(AgentClaude)
	if !def.ShouldRemoveOnEvent("SessionEnd") {
		t.Error("Claude ShouldRemoveOnEvent(SessionEnd) should be true")
	}
	if def.ShouldRemoveOnEvent("Stop") {
		t.Error("Claude ShouldRemoveOnEvent(Stop) should be false")
	}
}

func TestCodexShouldRemoveOnEvent(t *testing.T) {
	def, _ := GetAgent(AgentCodex)
	if def.ShouldRemoveOnEvent != nil {
		t.Error("Codex ShouldRemoveOnEvent should be nil")
	}
}

func TestClaudeNotifyTitle(t *testing.T) {
	def, _ := GetAgent(AgentClaude)
	if got := def.NotifyTitle("Notification"); got == "" {
		t.Error("Claude NotifyTitle(Notification) should not be empty")
	}
	if got := def.NotifyTitle("Stop"); got == "" {
		t.Error("Claude NotifyTitle(Stop) should not be empty")
	}
	if got := def.NotifyTitle("PreToolUse"); got != "" {
		t.Errorf("Claude NotifyTitle(PreToolUse) = %q, want empty", got)
	}
}

func TestCodexNotifyTitle(t *testing.T) {
	def, _ := GetAgent(AgentCodex)
	if got := def.NotifyTitle("Stop"); got == "" {
		t.Error("Codex NotifyTitle(Stop) should not be empty")
	}
	if got := def.NotifyTitle("SessionStart"); got != "" {
		t.Errorf("Codex NotifyTitle(SessionStart) = %q, want empty", got)
	}
}
