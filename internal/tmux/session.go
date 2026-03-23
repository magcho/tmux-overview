package tmux

import "time"

// PaneStatus represents the current state of a tmux pane.
type PaneStatus int

const (
	StatusUnknown PaneStatus = iota
	StatusRunning
	StatusDone
	StatusError
	StatusIdle
)

func (s PaneStatus) String() string {
	switch s {
	case StatusRunning:
		return "🤖 Running"
	case StatusDone:
		return "✅ Done"
	case StatusError:
		return "❌ Error"
	case StatusIdle:
		return "💤 Idle"
	default:
		return "❓ Unknown"
	}
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
	ID            string // e.g. %23
	Index         int
	CWD           string
	PID           int
	Width         int
	Height        int
	Active        bool
	Status        PaneStatus
	Duration      time.Duration
	Preview       []string
	SessionName   string
	WindowIndex   int
	WindowName    string
	WindowActive  bool
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
