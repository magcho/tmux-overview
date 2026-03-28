package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PaneCoord identifies a tmux pane by session name, window index, and pane index.
type PaneCoord struct {
	SessionName string
	WindowIndex int
	PaneIndex   int
}

func (c PaneCoord) Key() string {
	return fmt.Sprintf("%s:%d:%d", c.SessionName, c.WindowIndex, c.PaneIndex)
}

// ScanClaudePaneFiles reads /tmp/claude-notify-pane-* files and returns
// a map of pane coordinate keys to the most recent file modification time.
// The mtime represents when the hook last fired for that pane (≈ when Claude finished).
// Returns nil if no files are found.
func ScanClaudePaneFiles() map[string]time.Time {
	matches, err := filepath.Glob("/tmp/claude-notify-pane-*")
	if err != nil || len(matches) == 0 {
		return nil
	}

	coordTimes := make(map[string]time.Time)
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mtime := info.ModTime()

		coords, err := parseClaudePaneFile(path)
		if err != nil {
			continue
		}
		for _, c := range coords {
			key := c.Key()
			if existing, ok := coordTimes[key]; !ok || mtime.After(existing) {
				coordTimes[key] = mtime
			}
		}
	}

	if len(coordTimes) == 0 {
		return nil
	}
	return coordTimes
}

// parseClaudePaneFile parses a single pane info file.
//
// File format (4 lines for single-pane windows):
//
//	session_name
//	window_index
//	pane_index
//	/socket/path
//
// For multi-pane windows, list-panes returns multiple rows, producing:
//
//	session1
//	session2
//	window1
//	window2
//	pane1
//	pane2
//	/socket/path
func parseClaudePaneFile(path string) ([]PaneCoord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 4 {
		return nil, fmt.Errorf("too few lines: %d", len(lines))
	}

	// Find socket line (starts with /)
	socketIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "/") {
			socketIdx = i
			break
		}
	}
	if socketIdx < 3 {
		return nil, fmt.Errorf("socket line not found or too few data lines")
	}

	dataLines := lines[:socketIdx]
	if len(dataLines)%3 != 0 {
		return nil, fmt.Errorf("data lines not divisible by 3: %d", len(dataLines))
	}

	n := len(dataLines) / 3
	var coords []PaneCoord
	for i := 0; i < n; i++ {
		session := strings.TrimSpace(dataLines[i])
		window, _ := strconv.Atoi(strings.TrimSpace(dataLines[n+i]))
		pane, _ := strconv.Atoi(strings.TrimSpace(dataLines[2*n+i]))
		coords = append(coords, PaneCoord{
			SessionName: session,
			WindowIndex: window,
			PaneIndex:   pane,
		})
	}
	return coords, nil
}

// LookupPaneTime returns the hook mtime for a pane, or zero time if not found.
func LookupPaneTime(p Pane, coordTimes map[string]time.Time) time.Time {
	if coordTimes == nil {
		return time.Time{}
	}
	key := fmt.Sprintf("%s:%d:%d", p.SessionName, p.WindowIndex, p.Index)
	return coordTimes[key]
}
