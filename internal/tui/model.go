package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/tmux"
)

type focusPanel int

const (
	focusPaneList focusPanel = iota
	focusSessions
)

type tickMsg time.Time

type panesMsg struct {
	panes    []tmux.Pane
	sessions []string
	err      error
}

type Model struct {
	client   tmux.Client
	detector *tmux.StatusDetector
	cfg      config.Config

	allPanes     []tmux.Pane
	sessionNames []string

	// Cursors
	paneCursor    int
	sessionCursor int // 0 = "All", 1+ = session index

	// Focus
	focus focusPanel

	// Filter
	filterMode bool
	filterText string

	// Session filter (selected session name, empty = all)
	selectedSession string

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

// WithSessionFilter returns a copy with the session filter pre-set (from -s flag).
func (m Model) WithSessionFilter(session string) Model {
	m.selectedSession = session
	return m
}

// WithFilterText returns a copy with the text filter pre-set (from --filter flag).
func (m Model) WithFilterText(text string) Model {
	m.filterText = text
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchPanes(m.client, m.detector, m.cfg.Display.PreviewLines),
		tickCmd(time.Duration(m.cfg.Display.Interval) * time.Second),
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

		sessionSet := make(map[string]bool)
		var sessionNames []string

		for i := range panes {
			// Track unique session names in order
			if !sessionSet[panes[i].SessionName] {
				sessionSet[panes[i].SessionName] = true
				sessionNames = append(sessionNames, panes[i].SessionName)
			}

			lines, captureErr := c.CapturePaneContent(panes[i].ID, previewLines)
			if captureErr != nil {
				panes[i].Status = tmux.StatusUnknown
				continue
			}
			panes[i].Preview = lines
			panes[i].Status = detector.Detect(lines)
		}

		return panesMsg{panes: panes, sessions: sessionNames}
	}
}

// visiblePanes returns panes filtered by session selection and text filter.
func (m Model) visiblePanes() []tmux.Pane {
	panes := m.allPanes

	// Filter by selected session
	if m.selectedSession != "" {
		var filtered []tmux.Pane
		for _, p := range panes {
			if p.SessionName == m.selectedSession {
				filtered = append(filtered, p)
			}
		}
		panes = filtered
	}

	// Filter by text
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

// updateRunningDurations updates the running start times and sets Duration on panes.
func (m *Model) updateRunningDurations() {
	now := time.Now()
	activePanes := make(map[string]bool)

	for i := range m.allPanes {
		p := &m.allPanes[i]
		if p.Status == tmux.StatusRunning {
			activePanes[p.ID] = true
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
		if !activePanes[id] {
			delete(m.runningStartTimes, id)
		}
	}
}
