package tui

import (
	"strings"

	"github.com/magcho/tmux-overview/internal/tmux"
)

// filterPanes returns panes matching all space-separated words (AND condition).
func filterPanes(panes []tmux.Pane, query string) []tmux.Pane {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return panes
	}

	var result []tmux.Pane
	for _, p := range panes {
		searchText := strings.ToLower(strings.Join([]string{
			p.SessionName,
			p.WindowName,
			p.CWD,
			p.Status.String(),
			strings.Join(p.Preview, " "),
		}, " "))

		match := true
		for _, w := range words {
			if !strings.Contains(searchText, w) {
				match = false
				break
			}
		}
		if match {
			result = append(result, p)
		}
	}
	return result
}
