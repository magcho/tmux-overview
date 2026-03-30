package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/magcho/tmux-overview/internal/state"
)

// hookInput represents the common fields from Claude Code hook stdin JSON.
type hookInput struct {
	SessionID        string `json:"session_id"`
	CWD              string `json:"cwd"`
	HookEventName    string `json:"hook_event_name"`
	NotificationType string `json:"notification_type"`
}

// tmuxPaneInfo holds pane metadata fetched from tmux.
type tmuxPaneInfo struct {
	PaneID      string
	SessionName string
	WindowIndex int
	PaneIndex   int
}

// HandleEvent processes a Claude Code hook event and updates the pane state file.
func HandleEvent(eventType string, stdin io.Reader, store *state.Store) error {
	var input hookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		// stdin may be empty for some events; treat as empty input
		input = hookInput{}
	}

	paneInfo, err := getTmuxPaneInfo()
	if err != nil {
		// If not running inside tmux, exit silently (don't block Claude Code)
		fmt.Fprintf(os.Stderr, "tov hook: %v\n", err)
		return nil
	}

	now := time.Now()

	// SessionEnd: remove state file and return
	if eventType == "SessionEnd" {
		return store.Remove(paneInfo.PaneID)
	}

	// Load existing state or create new
	ps, exists, err := store.Read(paneInfo.PaneID)
	if err != nil {
		// Corrupted file; start fresh
		exists = false
	}

	if !exists {
		ps = state.PaneState{
			PaneID:          paneInfo.PaneID,
			SessionName:     paneInfo.SessionName,
			WindowIndex:     paneInfo.WindowIndex,
			PaneIndex:       paneInfo.PaneIndex,
			PID:             os.Getppid(),
			StatusChangedAt: now,
		}
	}

	// Always update pane metadata (may have changed if tmux renamed session, etc.)
	ps.SessionName = paneInfo.SessionName
	ps.WindowIndex = paneInfo.WindowIndex
	ps.PaneIndex = paneInfo.PaneIndex

	// Determine new status
	newStatus := applyStatusTransition(eventType, input, ps.Status)

	// Update status_changed_at only if status actually changed
	if newStatus != ps.Status {
		ps.Status = newStatus
		ps.StatusChangedAt = now
	}

	// Always update last event
	ps.LastEvent = eventType
	ps.LastEventAt = now

	// Store notification message for waiting status
	if eventType == "Notification" {
		ps.Message = input.NotificationType
	} else {
		ps.Message = ""
	}

	return store.Write(ps)
}

// applyStatusTransition determines the new status based on event type.
func applyStatusTransition(eventType string, input hookInput, current state.Status) state.Status {
	switch eventType {
	case "SessionStart":
		return state.StatusRegistered

	case "UserPromptSubmit":
		return state.StatusRunning

	case "PreToolUse":
		return state.StatusRunning

	case "Notification":
		switch input.NotificationType {
		case "permission_prompt", "elicitation_dialog":
			return state.StatusWaiting
		case "idle_prompt":
			return state.StatusDone
		default:
			return state.StatusWaiting
		}

	case "Stop":
		return state.StatusDone

	default:
		// Unknown event type; don't change status
		return current
	}
}

// getTmuxPaneInfo reads the current tmux pane information.
func getTmuxPaneInfo() (tmuxPaneInfo, error) {
	paneID := os.Getenv("TMUX_PANE")
	if paneID == "" {
		return tmuxPaneInfo{}, fmt.Errorf("TMUX_PANE not set (not running inside tmux?)")
	}

	// Get session name, window index, pane index from tmux
	out, err := exec.Command("tmux", "display-message", "-t", paneID, "-p",
		"#{session_name}\t#{window_index}\t#{pane_index}").Output()
	if err != nil {
		return tmuxPaneInfo{}, fmt.Errorf("tmux display-message: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(parts) != 3 {
		return tmuxPaneInfo{}, fmt.Errorf("unexpected tmux output: %q", string(out))
	}

	windowIdx, _ := strconv.Atoi(parts[1])
	paneIdx, _ := strconv.Atoi(parts[2])

	return tmuxPaneInfo{
		PaneID:      paneID,
		SessionName: parts[0],
		WindowIndex: windowIdx,
		PaneIndex:   paneIdx,
	}, nil
}
