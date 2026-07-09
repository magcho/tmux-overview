package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/magcho/tmux-overview/internal/tmux"
)

func TestHandleKeyWrapsUpFromFirstPaneToLast(t *testing.T) {
	m := testModel(t, 120, 24)
	m.paneCursor = 0

	updated, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	got := updated.(Model).paneCursor
	want := len(m.visiblePanes()) - 1

	if got != want {
		t.Fatalf("paneCursor = %d, want %d", got, want)
	}
}

func TestHandleKeyWrapsDownFromLastPaneToFirst(t *testing.T) {
	m := testModel(t, 120, 24)
	m.paneCursor = len(m.visiblePanes()) - 1

	updated, _ := m.handleKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	got := updated.(Model).paneCursor

	if got != 0 {
		t.Fatalf("paneCursor = %d, want 0", got)
	}
}

func TestUpdateInitialCursorSelectsMostRecentAttentionPane(t *testing.T) {
	m := testModel(t, 120, 24)
	m.originPaneID = "%2"
	panes := []tmux.Pane{
		{ID: "%1", Status: tmux.StatusRunning, Duration: 5 * time.Second},
		{ID: "%2", Status: tmux.StatusDone, Duration: 25 * time.Second},
		{ID: "%3", Status: tmux.StatusWaiting, Duration: 8 * time.Second},
		{ID: "%4", Status: tmux.StatusDone, Duration: 45 * time.Second},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 2 {
		t.Fatalf("paneCursor = %d, want 2", got)
	}
}

func TestUpdateInitialCursorSelectsOriginPaneWithoutRecentAttention(t *testing.T) {
	m := testModel(t, 120, 24)
	m.originPaneID = "%3"
	panes := []tmux.Pane{
		{ID: "%1", Status: tmux.StatusRunning, Duration: 5 * time.Second},
		{ID: "%2", Status: tmux.StatusDone, Duration: 31 * time.Second},
		{ID: "%3", Status: tmux.StatusWaiting, Duration: 45 * time.Second},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 2 {
		t.Fatalf("paneCursor = %d, want 2", got)
	}
}

func TestUpdateInitialCursorFallsBackToFirstPane(t *testing.T) {
	m := testModel(t, 120, 24)
	panes := []tmux.Pane{
		{ID: "%1", Status: tmux.StatusRunning, Duration: 5 * time.Second},
		{ID: "%2", Status: tmux.StatusDone, Duration: 31 * time.Second},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 0 {
		t.Fatalf("paneCursor = %d, want 0", got)
	}
}

func TestUpdateDoesNotResetInitializedCursor(t *testing.T) {
	m := testModel(t, 120, 24)
	panes := []tmux.Pane{
		{ID: "%1", Status: tmux.StatusDone, Duration: 5 * time.Second},
		{ID: "%2", Status: tmux.StatusDone, Duration: 10 * time.Second},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	m = updated.(Model)
	m.paneCursor = 1

	updated, _ = m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 1 {
		t.Fatalf("paneCursor = %d, want 1", got)
	}
}
