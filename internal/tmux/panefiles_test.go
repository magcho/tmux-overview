package tmux

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseClaudePaneFile_SinglePane(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pane-1234")
	os.WriteFile(path, []byte("8\n3\n0\n/private/tmp/tmux-501/default\n"), 0644)

	coords, err := parseClaudePaneFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coords) != 1 {
		t.Fatalf("expected 1 coord, got %d", len(coords))
	}
	if coords[0].SessionName != "8" || coords[0].WindowIndex != 3 || coords[0].PaneIndex != 0 {
		t.Errorf("unexpected coord: %+v", coords[0])
	}
}

func TestParseClaudePaneFile_MultiPane(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pane-5678")
	os.WriteFile(path, []byte("8\n8\n2\n2\n0\n1\n/private/tmp/tmux-501/default\n"), 0644)

	coords, err := parseClaudePaneFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(coords) != 2 {
		t.Fatalf("expected 2 coords, got %d", len(coords))
	}
	if coords[0].Key() != "8:2:0" {
		t.Errorf("coord[0]: got %s, want 8:2:0", coords[0].Key())
	}
	if coords[1].Key() != "8:2:1" {
		t.Errorf("coord[1]: got %s, want 8:2:1", coords[1].Key())
	}
}

func TestParseClaudePaneFile_InvalidFormat(t *testing.T) {
	dir := t.TempDir()

	// Too few lines
	path := filepath.Join(dir, "short")
	os.WriteFile(path, []byte("8\n3\n"), 0644)
	if _, err := parseClaudePaneFile(path); err == nil {
		t.Error("expected error for too few lines")
	}

	// No socket line
	path = filepath.Join(dir, "nosocket")
	os.WriteFile(path, []byte("8\n3\n0\nnotasocket\n"), 0644)
	if _, err := parseClaudePaneFile(path); err == nil {
		t.Error("expected error for missing socket")
	}
}

func TestLookupPaneTime(t *testing.T) {
	now := time.Now()
	coordTimes := map[string]time.Time{
		"8:2:0": now.Add(-30 * time.Second),
		"3:1:0": now.Add(-5 * time.Minute),
	}

	p1 := Pane{SessionName: "8", WindowIndex: 2, Index: 0}
	if got := LookupPaneTime(p1, coordTimes); got.IsZero() {
		t.Error("expected non-zero time for 8:2:0")
	}

	p2 := Pane{SessionName: "7", WindowIndex: 1, Index: 0}
	if got := LookupPaneTime(p2, coordTimes); !got.IsZero() {
		t.Error("expected zero time for 7:1:0")
	}

	// nil map should return zero
	if got := LookupPaneTime(p1, nil); !got.IsZero() {
		t.Error("expected zero time for nil map")
	}
}
