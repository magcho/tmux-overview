package tmux

import "time"

// PaneStatus represents the current state of a tmux pane.
type PaneStatus int

const (
	StatusUnknown PaneStatus = iota
	StatusRegistered
	StatusRunning
	StatusDone
	StatusWaiting
	StatusError
	StatusIdle
)

func (s PaneStatus) String() string {
	switch s {
	case StatusRegistered:
		return "📋 Registered"
	case StatusRunning:
		return "🤖 Running"
	case StatusDone:
		return "✅ Done"
	case StatusWaiting:
		return "⏸ Waiting"
	case StatusError:
		return "❌ Error"
	case StatusIdle:
		return "💤 Idle"
	default:
		return "❓ Unknown"
	}
}

// JapaneseLabel returns the status label in Japanese.
func (s PaneStatus) JapaneseLabel() string {
	switch s {
	case StatusRegistered:
		return "📋 起動中"
	case StatusRunning:
		return "🤖 処理中"
	case StatusDone:
		return "⏳ 返答待ち"
	case StatusWaiting:
		return "⏸ 確認待ち"
	case StatusError:
		return "❌ エラー"
	default:
		return "❓ 不明"
	}
}

// Label returns the status label in the specified language ("ja" for Japanese, default English).
func (s PaneStatus) Label(lang string) string {
	if lang == "ja" {
		return s.JapaneseLabel()
	}
	return s.String()
}

type Session struct {
	Name    string
	Windows []Window
}

type Window struct {
	Index  int
	Name   string
	Panes  []Pane
	Active bool
}

type Pane struct {
	ID           string // e.g. %23
	Index        int
	CWD          string
	PID          int
	Width        int
	Height       int
	Active       bool
	Status       PaneStatus
	Duration     time.Duration // Time since status changed
	Preview      []string
	Message      string // Notification message (for waiting status)
	SessionName  string
	WindowIndex  int
	WindowName   string
	WindowActive bool
}

// FlatLabel returns a display string like "work:1:biwa-frontend"
func (p Pane) FlatLabel() string {
	return p.SessionName + ":" + itoa(p.WindowIndex) + ":" + p.WindowName
}

func itoa(i int) string {
	if i < 0 {
		return "-" + itoa(-i)
	}
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}
