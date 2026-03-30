package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now().Truncate(time.Second)
	ps := PaneState{
		PaneID:          "%5",
		SessionName:     "work",
		WindowIndex:     1,
		PaneIndex:       0,
		PID:             12345,
		Status:          StatusRunning,
		StatusChangedAt: now,
		LastEvent:       "PreToolUse",
		LastEventAt:     now,
		Message:         "",
	}

	if err := store.Write(ps); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, exists, err := store.Read("%5")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !exists {
		t.Fatal("expected state file to exist")
	}
	if got.PaneID != "%5" {
		t.Errorf("PaneID = %q, want %%5", got.PaneID)
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, StatusRunning)
	}
	if got.PID != 12345 {
		t.Errorf("PID = %d, want 12345", got.PID)
	}
	if got.SessionName != "work" {
		t.Errorf("SessionName = %q, want %q", got.SessionName, "work")
	}
}

func TestReadNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	_, exists, err := store.Read("%99")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if exists {
		t.Error("expected state file to not exist")
	}
}

func TestListAll(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now().Truncate(time.Second)
	for _, id := range []string{"%1", "%2", "%3"} {
		ps := PaneState{
			PaneID:          id,
			Status:          StatusRunning,
			StatusChangedAt: now,
			LastEventAt:     now,
		}
		if err := store.Write(ps); err != nil {
			t.Fatalf("Write %s: %v", id, err)
		}
	}

	states, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(states) != 3 {
		t.Fatalf("ListAll returned %d states, want 3", len(states))
	}
}

func TestListAllEmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	states, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("ListAll returned %d states, want 0", len(states))
	}
}

func TestListAllNonExistentDir(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nonexistent"))

	states, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if states != nil {
		t.Fatalf("ListAll returned %v, want nil", states)
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	ps := PaneState{
		PaneID:      "%10",
		Status:      StatusDone,
		LastEventAt: time.Now(),
	}
	if err := store.Write(ps); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := store.Remove("%10"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, exists, err := store.Read("%10")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if exists {
		t.Error("expected state file to be removed")
	}
}

func TestRemoveNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if err := store.Remove("%99"); err != nil {
		t.Fatalf("Remove non-existent: %v", err)
	}
}

func TestRemoveStale(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	now := time.Now()
	for _, id := range []string{"%1", "%2", "%3"} {
		ps := PaneState{
			PaneID:      id,
			Status:      StatusRunning,
			LastEventAt: now,
		}
		if err := store.Write(ps); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	// Only %1 and %3 are live
	live := map[string]bool{"%1": true, "%3": true}
	removed := store.RemoveStale(live)
	if removed != 1 {
		t.Errorf("RemoveStale removed %d, want 1", removed)
	}

	// %2 should be gone
	_, exists, _ := store.Read("%2")
	if exists {
		t.Error("expected %2 to be removed")
	}

	// %1 and %3 should still exist
	_, exists1, _ := store.Read("%1")
	_, exists3, _ := store.Read("%3")
	if !exists1 || !exists3 {
		t.Error("expected %1 and %3 to still exist")
	}
}

func TestPaneFilename(t *testing.T) {
	tests := []struct {
		paneID   string
		expected string
	}{
		{"%5", "_5.json"},
		{"%23", "_23.json"},
		{"%100", "_100.json"},
	}
	for _, tt := range tests {
		got := paneFilename(tt.paneID)
		if got != tt.expected {
			t.Errorf("paneFilename(%q) = %q, want %q", tt.paneID, got, tt.expected)
		}
	}
}

func TestPaneIDFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"_5.json", "%5"},
		{"_23.json", "%23"},
	}
	for _, tt := range tests {
		got := paneIDFromFilename(tt.filename)
		if got != tt.expected {
			t.Errorf("paneIDFromFilename(%q) = %q, want %q", tt.filename, got, tt.expected)
		}
	}
}
