package state

import "time"

// Status represents the current state of a Claude Code session in a tmux pane.
type Status string

const (
	StatusRegistered Status = "registered" // Session started, not yet active
	StatusRunning    Status = "running"    // Actively processing
	StatusWaiting    Status = "waiting"    // Needs user input (permission prompt)
	StatusDone       Status = "done"       // Task completed
)

// PaneState holds the hook-derived state for a single tmux pane.
type PaneState struct {
	PaneID          string    `json:"pane_id"`
	SessionName     string    `json:"session_name"`
	WindowIndex     int       `json:"window_index"`
	PaneIndex       int       `json:"pane_index"`
	PID             int       `json:"pid"`
	Status          Status    `json:"status"`
	StatusChangedAt time.Time `json:"status_changed_at"`
	LastEvent       string    `json:"last_event"`
	LastEventAt     time.Time `json:"last_event_at"`
	Message         string    `json:"message,omitempty"`
}
