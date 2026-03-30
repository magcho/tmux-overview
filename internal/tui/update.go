package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		interval := time.Duration(m.cfg.Display.Interval) * time.Second
		return m, tea.Batch(
			fetchPanes(m.client, m.store, m.cfg.Display.PreviewLines),
			tickCmd(interval),
		)

	case panesMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.allPanes = msg.panes

		// Keep cursor in bounds
		visible := m.visiblePanes()
		if m.paneCursor >= len(visible) {
			m.paneCursor = max(0, len(visible)-1)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterMode {
		return m.handleFilterKey(msg)
	}

	action := parseKey(msg)

	switch action {
	case keyQuit:
		m.quitting = true
		return m, tea.Quit

	case keyUp:
		if m.paneCursor > 0 {
			m.paneCursor--
		}

	case keyDown:
		visible := m.visiblePanes()
		if m.paneCursor < len(visible)-1 {
			m.paneCursor++
		}

	case keyEnter:
		visible := m.visiblePanes()
		if len(visible) > 0 && m.paneCursor < len(visible) {
			p := visible[m.paneCursor]
			m.jumpPane = &p
			m.quitting = true
			return m, tea.Quit
		}

	case keyFilter:
		m.filterMode = true

	case keyRefresh:
		return m, fetchPanes(m.client, m.store, m.cfg.Display.PreviewLines)

	case keySpace:
		m.previewExpanded = !m.previewExpanded

	case keyEscape:
		if m.filterText != "" {
			m.filterText = ""
			m.paneCursor = 0
		}
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterText = ""
		m.paneCursor = 0
		return m, nil
	case "enter":
		m.filterMode = false
		return m, nil
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.paneCursor = 0
		}
		return m, nil
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.filterText += msg.String()
			m.paneCursor = 0
		}
		return m, nil
	}
}
