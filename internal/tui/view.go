package tui

import (
	"fmt"
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

	// Session sidebar styles
	sidebarTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("75")).
				PaddingBottom(1)

	sessionSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("229")).
				Background(lipgloss.Color("57"))

	sessionNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	sessionFocusedBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("75"))

	sessionUnfocusedBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	// Pane detail styles
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("75")).
				PaddingBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(10)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	detailBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238"))

	previewBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("241"))

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

	listFocusedBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("75"))

	listUnfocusedBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238"))

	// Status styles
	statusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	statusDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	statusIdleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusUnknownStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Footer
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

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

	var sections []string

	// === Header ===
	sections = append(sections, m.viewHeader(w))

	// === Middle: Sessions sidebar | Pane Detail ===
	sidebarWidth := 16
	detailWidth := w - sidebarWidth - 6 // account for borders and padding
	if detailWidth < 30 {
		detailWidth = 30
	}

	// Calculate heights for middle section
	middleHeight := 0
	if m.previewExpanded {
		middleHeight = min(h/3, 16)
	} else {
		middleHeight = 4
	}
	if middleHeight < 4 {
		middleHeight = 4
	}

	sidebar := m.viewSessionSidebar(sidebarWidth, middleHeight)
	detail := m.viewPaneDetail(detailWidth, middleHeight)
	middle := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, detail)
	sections = append(sections, middle)

	// === Pane List ===
	// Remaining height for pane list
	usedHeight := 3 + middleHeight + 4 + 2 // header + middle + footer + borders
	listHeight := max(h-usedHeight, 5)
	sections = append(sections, m.viewPaneList(w, listHeight))

	// === Footer ===
	sections = append(sections, m.viewFooter(w))

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
	stats := fmt.Sprintf("Sessions: %d  Panes: %d", len(m.sessionNames), len(m.allPanes))
	if len(visible) != len(m.allPanes) {
		stats += fmt.Sprintf("  (showing: %d)", len(visible))
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
		header += "\n" + errorMsgStyle.Render(fmt.Sprintf("  Error: %v", m.err))
	}

	return header
}

func (m Model) viewSessionSidebar(width, height int) string {
	var b strings.Builder
	b.WriteString(sidebarTitleStyle.Render("SESSIONS"))
	b.WriteString("\n")

	// "All" option
	allLabel := "  All"
	if m.sessionCursor == 0 {
		allLabel = "▶ All"
		allLabel = sessionSelectedStyle.Render(allLabel)
	} else {
		allLabel = sessionNormalStyle.Render(allLabel)
	}
	b.WriteString(allLabel)
	b.WriteString("\n")

	for i, name := range m.sessionNames {
		label := name
		if len(label) > width-4 {
			label = label[:width-7] + "…"
		}
		prefix := "  "
		if m.sessionCursor == i+1 {
			prefix = "▶ "
			line := prefix + label
			b.WriteString(sessionSelectedStyle.Render(line))
		} else {
			line := prefix + label
			b.WriteString(sessionNormalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	content := b.String()

	borderStyle := sessionUnfocusedBorder
	if m.focus == focusSessions {
		borderStyle = sessionFocusedBorder
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(content)
}

func (m Model) viewPaneDetail(width, height int) string {
	var b strings.Builder
	b.WriteString(detailTitleStyle.Render("PANE DETAIL"))
	b.WriteString("\n")

	pane := m.selectedPane()
	if pane == nil {
		b.WriteString(statsStyle.Render("  No pane selected"))
		return detailBorder.
			Width(width).
			Height(height).
			Render(b.String())
	}

	p := *pane

	// Detail fields
	writeDetail := func(label, value string) {
		b.WriteString(detailLabelStyle.Render(label))
		b.WriteString(detailValueStyle.Render(value))
		b.WriteString("\n")
	}

	writeDetail("Session", p.SessionName)
	writeDetail("Window", fmt.Sprintf("%d:%s", p.WindowIndex, p.WindowName))

	activeStr := ""
	if p.Active && p.WindowActive {
		activeStr = "  (active)"
	}
	writeDetail("Pane", p.ID+activeStr)
	writeDetail("CWD", abbreviateHome(p.CWD))
	writeDetail("Size", fmt.Sprintf("%dx%d", p.Width, p.Height))
	writeDetail("PID", fmt.Sprintf("%d", p.PID))

	statusStr := styledStatus(p.Status)
	if p.Status == tmux.StatusRunning && p.Duration > 0 {
		statusStr += fmt.Sprintf("  (%s)", formatDuration(p.Duration))
	}
	b.WriteString(detailLabelStyle.Render("Status"))
	b.WriteString(statusStr)
	b.WriteString("\n")

	// Preview
	if m.previewExpanded && len(p.Preview) > 0 {
		b.WriteString("\n")
		previewLines := filterEmptyTrailingLines(p.Preview)
		maxLines := m.cfg.Display.PreviewLines
		if len(previewLines) > maxLines {
			previewLines = previewLines[len(previewLines)-maxLines:]
		}
		previewContent := strings.Join(previewLines, "\n")
		previewWidth := width - 4
		if previewWidth < 20 {
			previewWidth = 20
		}
		b.WriteString(previewBorder.Width(previewWidth).Render(previewContent))
	}

	return detailBorder.
		Width(width).
		Height(height).
		Render(b.String())
}

func (m Model) viewPaneList(width, height int) string {
	var b strings.Builder
	b.WriteString(listTitleStyle.Render("PANE LIST"))
	b.WriteString("\n")

	// Column header
	colWidth := width - 6
	if colWidth < 60 {
		colWidth = 60
	}
	header := fmt.Sprintf("  %-30s %-7s %-14s %s", "SESSION:WIN", "PANE", "STATUS", "CWD")
	b.WriteString(listHeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(listHeaderStyle.Render(strings.Repeat("─", min(colWidth, 78))))
	b.WriteString("\n")

	panes := m.visiblePanes()

	if len(panes) == 0 {
		if len(m.allPanes) == 0 {
			b.WriteString(statsStyle.Render("  No tmux panes found."))
		} else {
			b.WriteString(statsStyle.Render("  No panes match filter."))
		}
		b.WriteString("\n")
	}

	// Scrolling
	listAvail := max(height-4, 3)
	start := 0
	if m.paneCursor >= listAvail {
		start = m.paneCursor - listAvail + 1
	}
	end := min(start+listAvail, len(panes))

	cwdMax := m.cfg.Display.CWDMaxLength
	for i := start; i < end; i++ {
		p := panes[i]
		line := formatPaneLine(p, cwdMax)

		if i == m.paneCursor {
			b.WriteString(paneSelectedStyle.Render("▶ " + line))
		} else if p.Status == tmux.StatusError {
			b.WriteString(paneErrorStyle.Render("  " + line))
		} else {
			b.WriteString(paneNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(panes) > listAvail {
		indicator := fmt.Sprintf("  (%d/%d)", m.paneCursor+1, len(panes))
		b.WriteString(statsStyle.Render(indicator))
		b.WriteString("\n")
	}

	borderStyle := listUnfocusedBorder
	if m.focus == focusPaneList {
		borderStyle = listFocusedBorder
	}

	return borderStyle.
		Width(width - 2).
		Height(height).
		Render(b.String())
}

func (m Model) viewFooter(_ int) string {
	if m.filterMode {
		return footerStyle.Render(" [Enter] 確定  [Esc] キャンセル")
	}
	line1 := " [↑↓/jk] 移動  [Enter] ジャンプ  [/] フィルター  [r] 更新"
	line2 := " [Tab] フォーカス切替  [Space] プレビュー展開/折畳  [1-9] セッション選択  [q] 終了"
	return footerStyle.Render(line1) + "\n" + footerStyle.Render(line2)
}

// === Helpers ===

func formatPaneLine(p tmux.Pane, cwdMax int) string {
	label := p.FlatLabel()
	if len(label) > 28 {
		label = label[:25] + "…"
	}

	statusStr := styledStatus(p.Status)
	if p.Status == tmux.StatusRunning && p.Duration > 0 {
		statusStr += fmt.Sprintf(" (%s)", formatDuration(p.Duration))
	}

	cwd := abbreviateHome(p.CWD)
	if cwdMax <= 0 {
		cwdMax = 40
	}
	if len(cwd) > cwdMax {
		cwd = "…" + cwd[len(cwd)-cwdMax+1:]
	}

	activeMarker := " "
	if p.Active && p.WindowActive {
		activeMarker = "*"
	}

	return fmt.Sprintf("%-28s %-6s%s %-14s %s", label, p.ID, activeMarker, statusStr, cwd)
}

func styledStatus(s tmux.PaneStatus) string {
	switch s {
	case tmux.StatusRunning:
		return statusRunningStyle.Render(s.String())
	case tmux.StatusDone:
		return statusDoneStyle.Render(s.String())
	case tmux.StatusError:
		return statusErrorStyle.Render(s.String())
	case tmux.StatusIdle:
		return statusIdleStyle.Render(s.String())
	default:
		return statusUnknownStyle.Render(s.String())
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
	for len(lines) > 0 {
		if strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}
	return lines
}
