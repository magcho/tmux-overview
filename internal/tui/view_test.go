package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/state"
	"github.com/magcho/tmux-overview/internal/tmux"
)

type mockClient struct{}

func (c *mockClient) ListSessions() ([]tmux.Session, error)                     { return nil, nil }
func (c *mockClient) ListAllPanes() ([]tmux.Pane, error)                        { return nil, nil }
func (c *mockClient) CapturePaneContent(id string, lines int) ([]string, error) { return nil, nil }
func (c *mockClient) SwitchToPane(p tmux.Pane) error                            { return nil }
func (c *mockClient) IsInsideTmux() bool                                        { return true }

func testPanes() []tmux.Pane {
	return []tmux.Pane{
		{ID: "%21", SessionName: "work", WindowIndex: 1, WindowName: "frontend", CWD: "/Users/test/src/frontend", Status: tmux.StatusRunning, Duration: 42 * time.Second, Width: 220, Height: 50, PID: 48291, Active: true, WindowActive: true, Preview: []string{"> Analyzing component tree...", "✓ Updated Button.tsx", "✓ Updated index.ts", "■ Writing tests..."}},
		{ID: "%22", SessionName: "work", WindowIndex: 1, WindowName: "frontend", CWD: "/Users/test/src/frontend", Status: tmux.StatusDone, Duration: 15 * time.Second, Width: 110, Height: 25, PID: 48292, Preview: []string{"✓ Task completed"}},
		{ID: "%31", SessionName: "dev", WindowIndex: 1, WindowName: "dashboard", CWD: "/Users/test/src/dashboard", Status: tmux.StatusRunning, Duration: 120 * time.Second, Width: 110, Height: 25, PID: 48301, Preview: []string{"Reading files...", "Thinking..."}},
		{ID: "%38", SessionName: "dev", WindowIndex: 2, WindowName: "infra", CWD: "/Users/test/src/infra", Status: tmux.StatusError, Width: 110, Height: 25, PID: 48302, Preview: []string{"Error: connection refused"}},
	}
}

func testModel(t *testing.T, width, height int) Model {
	t.Helper()
	cfg := config.DefaultConfig()
	store := state.NewStore(t.TempDir())
	m := NewModel(&mockClient{}, store, cfg)
	m.width = width
	m.height = height
	m.allPanes = testPanes()
	return m
}

func TestViewHeight24(t *testing.T) {
	m := testModel(t, 120, 24)
	checkViewFits(t, m, "24-line terminal")
}

func TestViewHeight40(t *testing.T) {
	m := testModel(t, 120, 40)
	checkViewFits(t, m, "40-line terminal (preview should appear)")
}

func TestViewHeight50(t *testing.T) {
	m := testModel(t, 120, 50)
	checkViewFits(t, m, "50-line terminal (large)")
}

func TestViewHeightSmall(t *testing.T) {
	m := testModel(t, 80, 15)
	checkViewFits(t, m, "15-line terminal (small)")
}

func TestViewPreviewCollapsed(t *testing.T) {
	m := testModel(t, 120, 24)
	m.previewExpanded = false
	checkViewFits(t, m, "24-line collapsed preview")
}

func TestVisiblePanesFiltering(t *testing.T) {
	m := testModel(t, 120, 24)
	visible := m.visiblePanes()
	// Should show Running(2) + Done(1) + Error(1) = 4 panes
	if len(visible) != 4 {
		t.Errorf("expected 4 visible agent panes, got %d", len(visible))
	}
	for _, p := range visible {
		if p.Status == tmux.StatusIdle || p.Status == tmux.StatusUnknown {
			t.Errorf("pane %s with status %s should be filtered out", p.ID, p.Status)
		}
	}
}

func TestJapaneseStatusLabels(t *testing.T) {
	tests := []struct {
		status tmux.PaneStatus
		want   string
	}{
		{tmux.StatusRunning, "🤖 処理中"},
		{tmux.StatusDone, "⏳ 返答待ち"},
		{tmux.StatusError, "❌ エラー"},
	}
	for _, tt := range tests {
		got := tt.status.JapaneseLabel()
		if got != tt.want {
			t.Errorf("JapaneseLabel() for %v: got %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusLabelLanguageSwitch(t *testing.T) {
	s := tmux.StatusRunning
	if got := s.Label("en"); got != "🤖 Running" {
		t.Errorf("Label(en): got %q, want %q", got, "🤖 Running")
	}
	if got := s.Label("ja"); got != "🤖 処理中" {
		t.Errorf("Label(ja): got %q, want %q", got, "🤖 処理中")
	}
}

func TestInferPaneStatusCodexPermissionPrompt(t *testing.T) {
	preview := []string{
		"• Calling",
		"  └ vibe_kanban.create_issue_relationship(...)",
		"Field 1/1",
		"Allow the vibe_kanban MCP server to run tool \"create_issue_relationship\"?",
		"enter to submit | esc to cancel",
	}

	got := inferPaneStatus("codex", tmux.StatusDone, preview)
	if got != tmux.StatusWaiting {
		t.Fatalf("inferPaneStatus() = %v, want %v", got, tmux.StatusWaiting)
	}
}

func TestInferPaneStatusCodexHookTrailer(t *testing.T) {
	preview := []string{
		"• Running Stop hook",
		"",
		"Stop hook (completed)",
	}

	got := inferPaneStatus("codex", tmux.StatusDone, preview)
	if got != tmux.StatusWaiting {
		t.Fatalf("inferPaneStatus() = %v, want %v", got, tmux.StatusWaiting)
	}
}

func TestInferPaneStatusIgnoresOldCodexHookLines(t *testing.T) {
	preview := []string{
		"• Running Stop hook",
		"Stop hook (completed)",
		"• AI に渡すときに迷わないよう、独立 issue のまま着手順を明示します。",
	}

	got := inferPaneStatus("codex", tmux.StatusRunning, preview)
	if got != tmux.StatusRunning {
		t.Fatalf("inferPaneStatus() = %v, want %v", got, tmux.StatusRunning)
	}
}

func TestInferPaneStatusNonCodexUnchanged(t *testing.T) {
	preview := []string{
		"Allow the vibe_kanban MCP server to run tool \"create_issue_relationship\"?",
		"enter to submit | esc to cancel",
	}

	got := inferPaneStatus("claude", tmux.StatusDone, preview)
	if got != tmux.StatusDone {
		t.Fatalf("inferPaneStatus() = %v, want %v", got, tmux.StatusDone)
	}
}

func checkViewFits(t *testing.T, m Model, label string) {
	t.Helper()
	output := m.View()
	lines := strings.Split(output, "\n")

	fmt.Printf("=== %s: %d lines for %d height ===\n", label, len(lines), m.height)
	for i, line := range lines {
		fmt.Printf("%3d: %s\n", i+1, line)
	}
	fmt.Println()

	if len(lines) > m.height {
		t.Errorf("[%s] View output has %d lines, exceeds terminal height %d (overflow: %d)", label, len(lines), m.height, len(lines)-m.height)
	}
}
