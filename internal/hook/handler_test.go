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

func stubPaneInfo(t *testing.T, paneID string) {
	t.Helper()
	orig := getPaneInfo
	getPaneInfo = func() (tmuxPaneInfo, error) {
		return tmuxPaneInfo{PaneID: paneID, SessionName: "s", WindowIndex: 1, PaneIndex: 0}, nil
	}
	t.Cleanup(func() { getPaneInfo = orig })
}

func clearCodexEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"CODEX_THREAD_ID", "CODEX_HOME", "CODEX_SANDBOX", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI"} {
		t.Setenv(k, "")
	}
}

func TestHandleEventRecordsSessionID(t *testing.T) {
	clearCodexEnv(t)
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	stubPaneInfo(t, "%7")
	store := state.NewStore(t.TempDir())

	err := HandleEvent("SessionStart", strings.NewReader(`{"session_id":"main"}`), store, config.NotifyConfig{})
	if err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	ps, exists, _ := store.Read("%7")
	if !exists {
		t.Fatal("expected state file to be written")
	}
	if ps.SessionID != "main" {
		t.Errorf("SessionID = %q, want %q", ps.SessionID, "main")
	}
}

func TestHandleEventIgnoresHeadlessClaude(t *testing.T) {
	clearCodexEnv(t)
	stubPaneInfo(t, "%7")
	store := state.NewStore(t.TempDir())

	if err := store.Write(state.PaneState{PaneID: "%7", SessionID: "main", Status: state.StatusDone}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// A headless `claude -p` spawned from the pane inherits TMUX_PANE.
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-cli")
	for _, ev := range []string{"SessionStart", "UserPromptSubmit", "Stop", "SessionEnd"} {
		if err := HandleEvent(ev, strings.NewReader(`{"session_id":"child"}`), store, config.NotifyConfig{}); err != nil {
			t.Fatalf("HandleEvent(%s): %v", ev, err)
		}
	}

	ps, exists, _ := store.Read("%7")
	if !exists {
		t.Fatal("headless session must not remove the pane's state file")
	}
	if ps.SessionID != "main" || ps.Status != state.StatusDone {
		t.Errorf("state was modified by headless session: %+v", ps)
	}
}

func TestHandleEventSessionEndFromOtherSessionKeepsState(t *testing.T) {
	clearCodexEnv(t)
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	stubPaneInfo(t, "%7")
	store := state.NewStore(t.TempDir())

	if err := store.Write(state.PaneState{PaneID: "%7", SessionID: "main", Status: state.StatusDone}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := HandleEvent("SessionEnd", strings.NewReader(`{"session_id":"other"}`), store, config.NotifyConfig{}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if _, exists, _ := store.Read("%7"); !exists {
		t.Error("SessionEnd from a different session must not remove the state file")
	}

	if err := HandleEvent("SessionEnd", strings.NewReader(`{"session_id":"main"}`), store, config.NotifyConfig{}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if _, exists, _ := store.Read("%7"); exists {
		t.Error("SessionEnd from the owning session should remove the state file")
	}
}

func TestHandleEventSessionEndWithoutRecordedSessionRemoves(t *testing.T) {
	clearCodexEnv(t)
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	stubPaneInfo(t, "%7")
	store := state.NewStore(t.TempDir())

	// State files written by older tov versions have no session_id.
	if err := store.Write(state.PaneState{PaneID: "%7", Status: state.StatusDone}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := HandleEvent("SessionEnd", strings.NewReader(`{"session_id":"main"}`), store, config.NotifyConfig{}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if _, exists, _ := store.Read("%7"); exists {
		t.Error("expected state file to be removed")
	}
}
