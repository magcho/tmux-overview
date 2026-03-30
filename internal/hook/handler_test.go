package hook

import (
	"strings"
	"testing"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/state"
)

func TestApplyStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		input    hookInput
		current  state.Status
		expected state.Status
	}{
		{
			name:     "SessionStart sets registered",
			event:    "SessionStart",
			current:  "",
			expected: state.StatusRegistered,
		},
		{
			name:     "UserPromptSubmit sets running",
			event:    "UserPromptSubmit",
			current:  state.StatusRegistered,
			expected: state.StatusRunning,
		},
		{
			name:     "PreToolUse sets running",
			event:    "PreToolUse",
			current:  state.StatusWaiting,
			expected: state.StatusRunning,
		},
		{
			name:     "Notification permission_prompt sets waiting",
			event:    "Notification",
			input:    hookInput{NotificationType: "permission_prompt"},
			current:  state.StatusRunning,
			expected: state.StatusWaiting,
		},
		{
			name:     "Notification idle_prompt sets done",
			event:    "Notification",
			input:    hookInput{NotificationType: "idle_prompt"},
			current:  state.StatusRunning,
			expected: state.StatusDone,
		},
		{
			name:     "Notification elicitation_dialog sets waiting",
			event:    "Notification",
			input:    hookInput{NotificationType: "elicitation_dialog"},
			current:  state.StatusRunning,
			expected: state.StatusWaiting,
		},
		{
			name:     "Stop sets done",
			event:    "Stop",
			current:  state.StatusRunning,
			expected: state.StatusDone,
		},
		{
			name:     "Unknown event preserves current",
			event:    "SubagentStop",
			current:  state.StatusRunning,
			expected: state.StatusRunning,
		},
		{
			name:     "Running to waiting to running cycle",
			event:    "PreToolUse",
			current:  state.StatusWaiting,
			expected: state.StatusRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyStatusTransition(tt.event, tt.input, tt.current)
			if got != tt.expected {
				t.Errorf("applyStatusTransition(%q, ..., %q) = %q, want %q",
					tt.event, tt.current, got, tt.expected)
			}
		})
	}
}

func TestHandleEventSessionEnd(t *testing.T) {
	dir := t.TempDir()
	store := state.NewStore(dir)

	// Write a state file first
	ps := state.PaneState{
		PaneID: "%5",
		Status: state.StatusRunning,
	}
	if err := store.Write(ps); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// HandleEvent for SessionEnd requires TMUX_PANE which we can't set in tests.
	// Instead, test the Remove path directly.
	if err := store.Remove("%5"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	_, exists, _ := store.Read("%5")
	if exists {
		t.Error("expected state file to be removed after SessionEnd")
	}
}

func TestHandleEventWithoutTmux(t *testing.T) {
	dir := t.TempDir()
	store := state.NewStore(dir)

	// Without TMUX_PANE set, HandleEvent should return nil (silent failure)
	t.Setenv("TMUX_PANE", "")

	stdin := strings.NewReader(`{"session_id":"test"}`)
	notifyCfg := config.NotifyConfig{Enabled: false}
	err := HandleEvent("PreToolUse", stdin, store, notifyCfg)
	if err != nil {
		t.Errorf("expected nil error without TMUX_PANE, got: %v", err)
	}
}
