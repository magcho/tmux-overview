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

func TestUpdateInitialCursorSelectsOriginPane(t *testing.T) {
	m := testModel(t, 120, 24)
	m.originPaneID = "%3"
	m.originSession = "dev"
	panes := []tmux.Pane{
		{ID: "%1", SessionName: "work", Status: tmux.StatusWaiting, Duration: 5 * time.Second},
		{ID: "%2", SessionName: "dev", Status: tmux.StatusRunning},
		{ID: "%3", SessionName: "dev", Status: tmux.StatusDone},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 2 {
		t.Fatalf("paneCursor = %d, want 2", got)
	}
}

func TestUpdateInitialCursorSelectsFirstPaneInOriginSession(t *testing.T) {
	m := testModel(t, 120, 24)
	m.originPaneID = "%99" // origin pane itself is not an agent pane
	m.originSession = "dev"
	panes := []tmux.Pane{
		{ID: "%1", SessionName: "work", Status: tmux.StatusWaiting, Duration: 5 * time.Second},
		{ID: "%2", SessionName: "dev", Status: tmux.StatusRunning},
		{ID: "%3", SessionName: "dev", Status: tmux.StatusDone},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 1 {
		t.Fatalf("paneCursor = %d, want 1", got)
	}
}

func TestUpdateInitialCursorFallsBackToTopWhenOriginSessionHasNoPanes(t *testing.T) {
	m := testModel(t, 120, 24)
	m.originSession = "other"
	panes := []tmux.Pane{
		{ID: "%1", SessionName: "work", Status: tmux.StatusRunning},
		{ID: "%2", SessionName: "dev", Status: tmux.StatusDone, Duration: 5 * time.Second},
	}

	updated, _ := m.Update(panesMsg{panes: panes})
	got := updated.(Model).paneCursor

	if got != 0 {
		t.Fatalf("paneCursor = %d, want 0", got)
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

func TestParseKeyForceRefresh(t *testing.T) {
	if got := parseKey(tea.KeyPressMsg(tea.Key{Code: 'R', Text: "R"})); got != keyForceRefresh {
		t.Errorf("parseKey(R) = %v, want keyForceRefresh", got)
	}
	if got := parseKey(tea.KeyPressMsg(tea.Key{Code: 'r', Text: "r"})); got != keyRefresh {
		t.Errorf("parseKey(r) = %v, want keyRefresh", got)
	}
}
