package tui

import tea "github.com/charmbracelet/bubbletea"

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
	keyTab
	keySpace
	keySession1
	keySession2
	keySession3
	keySession4
	keySession5
	keySession6
	keySession7
	keySession8
	keySession9
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
	case "tab":
		return keyTab
	case " ":
		return keySpace
	case "1":
		return keySession1
	case "2":
		return keySession2
	case "3":
		return keySession3
	case "4":
		return keySession4
	case "5":
		return keySession5
	case "6":
		return keySession6
	case "7":
		return keySession7
	case "8":
		return keySession8
	case "9":
		return keySession9
	}
	return keyNone
}

func sessionNumberFromKey(action keyAction) int {
	switch action {
	case keySession1:
		return 1
	case keySession2:
		return 2
	case keySession3:
		return 3
	case keySession4:
		return 4
	case keySession5:
		return 5
	case keySession6:
		return 6
	case keySession7:
		return 7
	case keySession8:
		return 8
	case keySession9:
		return 9
	}
	return 0
}
