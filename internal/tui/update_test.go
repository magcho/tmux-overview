package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
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
