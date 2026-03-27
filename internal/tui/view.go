package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/magcho/tmux-overview/internal/tmux"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	filterLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	// Pane detail styles
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("75"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(10)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	detailBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("75"))

	previewLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	// Pane list styles
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75"))

	listHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	paneSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57"))

	paneNormalStyle = lipgloss.NewStyle()

	paneErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	listBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("75"))

	// Status styles
	statusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	statusDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	statusUnknownStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Footer
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

const borderOverhead = 2 // top + bottom border lines

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	// Layout budget (vertical):
	//   header:  1 line
	//   detail:  middleInner + 2 (border)
	//   list:    listInner + 2 (border)
	//   footer:  1 line
	//   Total = middleInner + listInner + 6
	const fixedLines = 1 + borderOverhead + borderOverhead + 1 // header + 2 borders + footer

	available := h - fixedLines
	var middleInner int
	if m.previewExpanded {
		middleInner = available * 2 / 3
		if middleInner > 18 {
			middleInner = 18
		}
	} else {
		middleInner = min(8, available*2/3)
	}
	if middleInner < 3 {
		middleInner = 3
	}

	listInner := available - middleInner
	if listInner < 3 {
		listInner = 3
	}

	var sections []string

	// === Header ===
	sections = append(sections, m.viewHeader(w))

	// === Pane Detail (full width) ===
	detailWidth := w - borderOverhead
	if detailWidth < 30 {
		detailWidth = 30
	}
	sections = append(sections, m.viewPaneDetail(detailWidth, middleInner))

	// === Pane List ===
	sections = append(sections, m.viewPaneList(w, listInner))

	// === Footer ===
	sections = append(sections, m.viewFooter())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) viewHeader(width int) string {
	title := titleStyle.Render(" tov ")

	filterStr := ""
	if m.filterMode {
		filterStr = filterLabelStyle.Render("Filter: ") + m.filterText + "█"
	} else if m.filterText != "" {
		filterStr = filterLabelStyle.Render("Filter: ") + m.filterText
	}

	visible := m.visiblePanes()
	stats := fmt.Sprintf("Claude: %d panes", len(visible))
	if len(visible) != len(m.allPanes) {
		stats += fmt.Sprintf(" / %d total", len(m.allPanes))
	}
	statsStr := statsStyle.Render(stats)

	var parts []string
	parts = append(parts, title)
	if filterStr != "" {
		parts = append(parts, filterStr)
	}
	parts = append(parts, statsStr)

	header := strings.Join(parts, "  │  ")

	if m.err != nil {
		header += "  " + errorMsgStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return header
}

func (m Model) viewPaneDetail(width, innerHeight int) string {
	var lines []string
	lines = append(lines, detailTitleStyle.Render("PANE DETAIL"))

	pane := m.selectedPane()
	if pane == nil {
		lines = append(lines, statsStyle.Render("  No pane selected"))
		content := truncateLines(lines, innerHeight)
		return detailBorder.
			Width(width).
			Height(innerHeight).
			Render(content)
	}

	p := *pane
	lang := m.cfg.Display.Language

	addDetail := func(label, value string) {
		lines = append(lines, detailLabelStyle.Render(label)+detailValueStyle.Render(value))
	}

	addDetail("Session", p.SessionName)
	addDetail("Window", fmt.Sprintf("%d:%s", p.WindowIndex, p.WindowName))

	activeStr := ""
	if p.Active && p.WindowActive {
		activeStr = "  (active)"
	}
	addDetail("Pane", p.ID+activeStr)
	addDetail("CWD", abbreviateHome(p.CWD))
	addDetail("Size", fmt.Sprintf("%dx%d", p.Width, p.Height))
	addDetail("PID", fmt.Sprintf("%d", p.PID))

	statusStr := styledStatusLabel(p.Status, lang)
	if p.Status == tmux.StatusRunning && p.Duration > 0 {
		statusStr += fmt.Sprintf("  (%s)", formatDuration(p.Duration))
	}
	if p.Status == tmux.StatusDone && p.WaitDuration > 0 {
		statusStr += fmt.Sprintf("  (%s)", formatDuration(p.WaitDuration))
	}
	lines = append(lines, detailLabelStyle.Render("Status")+statusStr)

	// Preview (inline, no nested border)
	if m.previewExpanded && len(p.Preview) > 0 {
		previewContent := filterEmptyTrailingLines(p.Preview)
		remaining := innerHeight - len(lines) - 1
		if remaining > m.cfg.Display.PreviewLines {
			remaining = m.cfg.Display.PreviewLines
		}
		if remaining > 0 {
			lines = append(lines, previewLabelStyle.Render("Preview:"))
			if len(previewContent) > remaining {
				previewContent = previewContent[len(previewContent)-remaining:]
			}
			for _, pl := range previewContent {
				lines = append(lines, "  "+pl)
			}
		}
	}

	content := truncateLines(lines, innerHeight)

	return detailBorder.
		Width(width).
		Height(innerHeight).
		Render(content)
}

func (m Model) viewPaneList(width, innerHeight int) string {
	lang := m.cfg.Display.Language
	var lines []string
	lines = append(lines, listTitleStyle.Render("PANE LIST"))

	// Column header
	header := fmt.Sprintf("  %-24s %-16s %s", "DIRECTORY", "STATUS", "DURATION")
	lines = append(lines, listHeaderStyle.Render(header))
	colWidth := min(width-6, 60)
	if colWidth < 40 {
		colWidth = 40
	}
	lines = append(lines, listHeaderStyle.Render(strings.Repeat("─", colWidth)))

	panes := m.visiblePanes()

	if len(panes) == 0 {
		if len(m.allPanes) == 0 {
			lines = append(lines, statsStyle.Render("  No tmux panes found."))
		} else {
			lines = append(lines, statsStyle.Render("  No Claude panes found."))
		}
	}

	// Available rows for pane entries
	headerLines := 3 // title + column header + separator
	paneRows := innerHeight - headerLines
	if len(panes) > paneRows {
		paneRows-- // reserve 1 line for scroll indicator
	}
	if paneRows < 1 {
		paneRows = 1
	}

	// Scrolling
	start := 0
	if m.paneCursor >= paneRows {
		start = m.paneCursor - paneRows + 1
	}
	end := min(start+paneRows, len(panes))

	for i := start; i < end; i++ {
		p := panes[i]
		line := formatPaneLine(p, lang)

		if i == m.paneCursor {
			lines = append(lines, paneSelectedStyle.Render("▶ "+line))
		} else if p.Status == tmux.StatusError {
			lines = append(lines, paneErrorStyle.Render("  "+line))
		} else {
			lines = append(lines, paneNormalStyle.Render("  "+line))
		}
	}

	// Scroll indicator
	if len(panes) > paneRows {
		indicator := fmt.Sprintf("  (%d/%d)", m.paneCursor+1, len(panes))
		lines = append(lines, statsStyle.Render(indicator))
	}

	content := truncateLines(lines, innerHeight)

	return listBorder.
		Width(width - 2).
		Height(innerHeight).
		Render(content)
}

func (m Model) viewFooter() string {
	if m.filterMode {
		return footerStyle.Render(" [Enter] 確定  [Esc] キャンセル")
	}
	return footerStyle.Render(" [↑↓/jk] 移動  [Enter] ジャンプ  [/] フィルター  [Space] プレビュー展開/折畳  [r] 更新  [Esc] フィルタ解除  [q] 終了")
}

// === Helpers ===

func truncateLines(lines []string, maxLines int) string {
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func formatPaneLine(p tmux.Pane, lang string) string {
	dir := filepath.Base(p.CWD)
	if len(dir) > 22 {
		dir = dir[:19] + "…"
	}

	statusStr := styledStatusLabel(p.Status, lang)

	var durationStr string
	switch p.Status {
	case tmux.StatusRunning:
		if p.Duration > 0 {
			durationStr = formatDuration(p.Duration)
		} else {
			durationStr = "0s"
		}
	case tmux.StatusDone:
		if p.WaitDuration > 0 {
			durationStr = formatDuration(p.WaitDuration)
		} else {
			durationStr = "0s"
		}
	default:
		durationStr = "-"
	}

	return fmt.Sprintf("%-22s  %-16s  %s", dir, statusStr, durationStr)
}

func styledStatusLabel(s tmux.PaneStatus, lang string) string {
	label := s.Label(lang)
	switch s {
	case tmux.StatusRunning:
		return statusRunningStyle.Render(label)
	case tmux.StatusDone:
		return statusDoneStyle.Render(label)
	case tmux.StatusError:
		return statusErrorStyle.Render(label)
	default:
		return statusUnknownStyle.Render(label)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func abbreviateHome(path string) string {
	home := ""
	if len(path) > 6 && path[:6] == "/home/" {
		rest := path[6:]
		slashIdx := strings.Index(rest, "/")
		if slashIdx >= 0 {
			home = path[:6+slashIdx]
		}
	} else if len(path) > 7 && path[:7] == "/Users/" {
		rest := path[7:]
		slashIdx := strings.Index(rest, "/")
		if slashIdx >= 0 {
			home = path[:7+slashIdx]
		}
	}
	if home != "" {
		return "~" + path[len(home):]
	}
	return path
}

func filterEmptyTrailingLines(lines []string) []string {
	result := make([]string, len(lines))
	copy(result, lines)
	for len(result) > 0 {
		if strings.TrimSpace(result[len(result)-1]) == "" {
			result = result[:len(result)-1]
		} else {
			break
		}
	}
	return result
}
