package tui

import tea "charm.land/bubbletea/v2"

type keyAction int

const (
	keyNone keyAction = iota
	keyUp
	keyDown
	keyEnter
	keyFilter
	keyEscape
	keyRefresh
	keyQuit
	keySpace
)

func parseKey(msg tea.KeyMsg) keyAction {
	switch msg.String() {
	case "up", "k":
		return keyUp
	case "down", "j":
		return keyDown
	case "enter":
		return keyEnter
	case "/":
		return keyFilter
	case "esc":
		return keyEscape
	case "r":
		return keyRefresh
	case "q", "ctrl+c":
		return keyQuit
	case " ":
		return keySpace
	}
	return keyNone
}
