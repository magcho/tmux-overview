package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/magcho/tmux-overview/internal/codex"
	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/state"
	"github.com/magcho/tmux-overview/internal/tmux"
)

type tickMsg time.Time

type panesMsg struct {
	panes []tmux.Pane
	err   error
}

type Model struct {
	client tmux.Client
	store  *state.Store
	cfg    config.Config

	allPanes []tmux.Pane

	// Cursor
	paneCursor int

	// Filter
	filterMode bool
	filterText string

	// Preview
	previewExpanded bool

	// Terminal size
	width  int
	height int

	// Error
	err error

	// Exit state
	jumpPane *tmux.Pane
	quitting bool
}

func NewModel(client tmux.Client, store *state.Store, cfg config.Config) Model {
	return Model{
		client:          client,
		store:           store,
		cfg:             cfg,
		previewExpanded: true,
	}
}

func (m Model) JumpPane() *tmux.Pane {
	return m.jumpPane
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchPanes(m.client, m.store, m.cfg.Display.PreviewLines),
		tickCmd(time.Duration(m.cfg.Display.Interval)*time.Second),
	)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// mapStatus converts a state.Status to a tmux.PaneStatus.
func mapStatus(s state.Status) tmux.PaneStatus {
	switch s {
	case state.StatusRegistered:
		return tmux.StatusRegistered
	case state.StatusRunning:
		return tmux.StatusRunning
	case state.StatusWaiting:
		return tmux.StatusWaiting
	case state.StatusDone:
		return tmux.StatusDone
	default:
		return tmux.StatusUnknown
	}
}

func fetchPanes(c tmux.Client, store *state.Store, previewLines int) tea.Cmd {
	return func() tea.Msg {
		allPanes, err := c.ListAllPanes()
		if err != nil {
			return panesMsg{err: err}
		}

		// Read all hook state files
		states, _ := store.ListAll()
		stateMap := make(map[string]state.PaneState)
		for _, s := range states {
			stateMap[s.PaneID] = s
		}

		now := time.Now()
		var agentPanes []tmux.Pane

		// Build live pane set for stale cleanup
		livePaneIDs := make(map[string]bool)
		for _, p := range allPanes {
			livePaneIDs[p.ID] = true
		}

		for i := range allPanes {
			ps, isAgent := stateMap[allPanes[i].ID]
			if !isAgent {
				continue
			}

			allPanes[i].Status = mapStatus(ps.Status)
			allPanes[i].Duration = now.Sub(ps.StatusChangedAt)
			allPanes[i].Message = ps.Message
			allPanes[i].Agent = ps.Agent

			// Capture pane content for preview only (not for status detection)
			lines, captureErr := c.CapturePaneContent(allPanes[i].ID, previewLines)
			if captureErr == nil {
				allPanes[i].Preview = lines
				allPanes[i].Status = inferPaneStatus(allPanes[i].Agent, allPanes[i].Status, lines)
			}

			agentPanes = append(agentPanes, allPanes[i])
		}

		// Clean up stale state files for panes that no longer exist in tmux
		store.RemoveStale(livePaneIDs)

		return panesMsg{panes: agentPanes}
	}
}

func inferPaneStatus(agent string, current tmux.PaneStatus, preview []string) tmux.PaneStatus {
	if agent != "codex" || len(preview) == 0 {
		return current
	}

	if waiting, _ := codex.DetectWaiting(preview); waiting {
		return tmux.StatusWaiting
	}

	return current
}

// visiblePanes returns agent-active panes, optionally filtered by text.
func (m Model) visiblePanes() []tmux.Pane {
	var panes []tmux.Pane
	for _, p := range m.allPanes {
		switch p.Status {
		case tmux.StatusRegistered, tmux.StatusRunning, tmux.StatusDone, tmux.StatusWaiting, tmux.StatusError:
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
