package discover

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/magcho/tmux-overview/internal/state"
)

func writeSession(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func testSources(t *testing.T, alive map[int]bool, procs map[string][]Process) Sources {
	t.Helper()
	return Sources{
		ClaudeSessionsDir: t.TempDir(),
		TTYProcesses: func(tty string) ([]Process, error) {
			return procs[tty], nil
		},
		IsAlive:    func(pid int) bool { return alive[pid] },
		TmuxSocket: "/tmp/tmux-501/default",
		Now:        func() time.Time { return time.Unix(1700000000, 0) },
	}
}

func TestRestoreFromClaudeSessionRegistry(t *testing.T) {
	store := state.NewStore(t.TempDir())
	src := testSources(t, map[int]bool{100: true}, nil)
	writeSession(t, src.ClaudeSessionsDir, "100.json",
		`{"pid":100,"sessionId":"sess-a","kind":"interactive","entrypoint":"cli","status":"idle","tmux":"20:@37.%55","statusUpdatedAt":1699999000000}`)

	n, err := Restore(store, []Pane{{ID: "%55", SessionName: "20", WindowIndex: 1, TTY: "/dev/ttys001"}}, src)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if n != 1 {
		t.Fatalf("restored = %d, want 1", n)
	}
	ps, exists, _ := store.Read("%55")
	if !exists {
		t.Fatal("expected state file for %55")
	}
	if ps.Agent != "claude" || ps.SessionID != "sess-a" || ps.PID != 100 || ps.Status != state.StatusDone {
		t.Errorf("unexpected state: %+v", ps)
	}
	if !ps.StatusChangedAt.Equal(time.UnixMilli(1699999000000)) {
		t.Errorf("StatusChangedAt = %v", ps.StatusChangedAt)
	}
	if ps.SessionName != "20" || ps.WindowIndex != 1 || ps.TmuxSocket != "/tmp/tmux-501/default" {
		t.Errorf("unexpected pane metadata: %+v", ps)
	}
}

func TestRestoreMapsRegistryStatus(t *testing.T) {
	tests := []struct {
		status string
		want   state.Status
	}{
		{"idle", state.StatusDone},
		{"busy", state.StatusRunning},
		{"waiting", state.StatusWaiting},
		{"something-new", state.StatusRegistered},
	}
	for _, tt := range tests {
		store := state.NewStore(t.TempDir())
		src := testSources(t, map[int]bool{100: true}, nil)
		writeSession(t, src.ClaudeSessionsDir, "100.json",
			`{"pid":100,"sessionId":"s","kind":"interactive","entrypoint":"cli","status":"`+tt.status+`","tmux":"1:@1.%1"}`)
		if _, err := Restore(store, []Pane{{ID: "%1"}}, src); err != nil {
			t.Fatal(err)
		}
		ps, _, _ := store.Read("%1")
		if ps.Status != tt.want {
			t.Errorf("status %q -> %q, want %q", tt.status, ps.Status, tt.want)
		}
	}
}

func TestRestoreSkipsDeadAndNonInteractiveSessions(t *testing.T) {
	store := state.NewStore(t.TempDir())
	src := testSources(t, map[int]bool{200: true}, nil)
	writeSession(t, src.ClaudeSessionsDir, "100.json",
		`{"pid":100,"sessionId":"dead","kind":"interactive","entrypoint":"cli","status":"idle","tmux":"1:@1.%1"}`)
	writeSession(t, src.ClaudeSessionsDir, "200.json",
		`{"pid":200,"sessionId":"headless","kind":"interactive","entrypoint":"sdk-cli","status":"busy","tmux":"1:@1.%1"}`)

	n, err := Restore(store, []Pane{{ID: "%1"}}, src)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("restored = %d, want 0", n)
	}
}

func TestRestoreDoesNotOverwriteExistingState(t *testing.T) {
	store := state.NewStore(t.TempDir())
	if err := store.Write(state.PaneState{PaneID: "%55", SessionID: "hook", Status: state.StatusRunning}); err != nil {
		t.Fatal(err)
	}
	src := testSources(t, map[int]bool{100: true}, nil)
	writeSession(t, src.ClaudeSessionsDir, "100.json",
		`{"pid":100,"sessionId":"registry","kind":"interactive","entrypoint":"cli","status":"idle","tmux":"20:@37.%55"}`)

	n, err := Restore(store, []Pane{{ID: "%55"}}, src)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("restored = %d, want 0", n)
	}
	ps, _, _ := store.Read("%55")
	if ps.SessionID != "hook" {
		t.Errorf("existing state was overwritten: %+v", ps)
	}
}

func TestRestoreFallsBackToTTYProcesses(t *testing.T) {
	store := state.NewStore(t.TempDir())
	src := testSources(t, nil, map[string][]Process{
		"/dev/ttys001": {{PID: 10, Args: "/bin/zsh"}, {PID: 11, Args: "claude --resume abc"}},
		"/dev/ttys002": {{PID: 20, Args: "-zsh"}, {PID: 21, Args: "/opt/homebrew/bin/codex"}},
		"/dev/ttys003": {{PID: 30, Args: "/bin/zsh"}, {PID: 31, Args: "claude -p summarize"}},
		"/dev/ttys004": {{PID: 40, Args: "/bin/zsh"}, {PID: 41, Args: "vim claude.md"}},
	})
	panes := []Pane{
		{ID: "%1", TTY: "/dev/ttys001"},
		{ID: "%2", TTY: "/dev/ttys002"},
		{ID: "%3", TTY: "/dev/ttys003"},
		{ID: "%4", TTY: "/dev/ttys004"},
	}

	n, err := Restore(store, panes, src)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("restored = %d, want 2", n)
	}
	if ps, ok, _ := store.Read("%1"); !ok || ps.Agent != "claude" || ps.PID != 11 || ps.Status != state.StatusRegistered {
		t.Errorf("%%1: exists=%v state=%+v", ok, ps)
	}
	if ps, ok, _ := store.Read("%2"); !ok || ps.Agent != "codex" || ps.PID != 21 {
		t.Errorf("%%2: exists=%v state=%+v", ok, ps)
	}
	for _, id := range []string{"%3", "%4"} {
		if _, ok, _ := store.Read(id); ok {
			t.Errorf("%s should not be restored", id)
		}
	}
}
