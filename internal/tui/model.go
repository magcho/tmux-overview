package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/tmux"
)

type tickMsg time.Time

type panesMsg struct {
	panes []tmux.Pane
	err   error
}

type Model struct {
	client   tmux.Client
	detector *tmux.StatusDetector
	cfg      config.Config

	allPanes []tmux.Pane

	// Cursor
	paneCursor int

	// Filter
	filterMode bool
	filterText string

	// Preview
	previewExpanded bool

	// Running duration tracking: paneID -> first seen running time
	runningStartTimes map[string]time.Time

	// Terminal size
	width  int
	height int

	// Error
	err error

	// Exit state
	jumpPane *tmux.Pane
	quitting bool
}

func NewModel(client tmux.Client, detector *tmux.StatusDetector, cfg config.Config) Model {
	return Model{
		client:            client,
		detector:          detector,
		cfg:               cfg,
		previewExpanded:   true,
		runningStartTimes: make(map[string]time.Time),
	}
}

func (m Model) JumpPane() *tmux.Pane {
	return m.jumpPane
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchPanes(m.client, m.detector, m.cfg.Display.PreviewLines),
		tickCmd(time.Duration(m.cfg.Display.Interval)*time.Second),
	)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchPanes(c tmux.Client, detector *tmux.StatusDetector, previewLines int) tea.Cmd {
	return func() tea.Msg {
		panes, err := c.ListAllPanes()
		if err != nil {
			return panesMsg{err: err}
		}

		// Scan hook-generated pane files to identify Claude panes.
		// Returns map of pane coord key -> most recent hook mtime.
		claudeFiles := tmux.ScanClaudePaneFiles()

		now := time.Now()
		for i := range panes {
			hookTime := tmux.LookupPaneTime(panes[i], claudeFiles)

			if claudeFiles != nil && hookTime.IsZero() {
				// Not a Claude pane candidate — skip capture for efficiency
				panes[i].Status = tmux.StatusIdle
				continue
			}

			lines, captureErr := c.CapturePaneContent(panes[i].ID, previewLines)
			if captureErr != nil {
				panes[i].Status = tmux.StatusUnknown
				continue
			}
			panes[i].Preview = lines
			panes[i].Status = detector.Detect(lines)

			// For Done/Waiting panes, compute WaitDuration from hook file mtime
			if (panes[i].Status == tmux.StatusDone || panes[i].Status == tmux.StatusWaiting) && !hookTime.IsZero() {
				panes[i].WaitDuration = now.Sub(hookTime)
			}
		}

		return panesMsg{panes: panes}
	}
}

// visiblePanes returns only Claude-active panes (Running/Done/Error), optionally filtered by text.
func (m Model) visiblePanes() []tmux.Pane {
	var panes []tmux.Pane
	for _, p := range m.allPanes {
		if p.Status == tmux.StatusRunning || p.Status == tmux.StatusDone || p.Status == tmux.StatusWaiting || p.Status == tmux.StatusError {
			panes = append(panes, p)
		}
	}

	if m.filterText != "" {
		panes = filterPanes(panes, m.filterText)
	}

	return panes
}

// selectedPane returns the currently highlighted pane, or nil.
func (m Model) selectedPane() *tmux.Pane {
	visible := m.visiblePanes()
	if m.paneCursor >= 0 && m.paneCursor < len(visible) {
		p := visible[m.paneCursor]
		return &p
	}
	return nil
}

// updateDurations updates running start times and sets Duration on panes.
// WaitDuration for Done panes is computed from hook file mtime in fetchPanes.
func (m *Model) updateDurations() {
	now := time.Now()
	activeRunning := make(map[string]bool)

	for i := range m.allPanes {
		p := &m.allPanes[i]
		if p.Status == tmux.StatusRunning {
			activeRunning[p.ID] = true
			if startTime, exists := m.runningStartTimes[p.ID]; exists {
				p.Duration = now.Sub(startTime)
			} else {
				m.runningStartTimes[p.ID] = now
				p.Duration = 0
			}
		} else {
			p.Duration = 0
		}
	}

	// Clean up panes that are no longer running
	for id := range m.runningStartTimes {
		if !activeRunning[id] {
			delete(m.runningStartTimes, id)
		}
	}
}
