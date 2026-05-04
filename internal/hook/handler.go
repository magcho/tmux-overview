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

	"github.com/magcho/tmux-overview/internal/codex"
	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/state"
)

// hookInput represents the common fields from agent hook stdin JSON.
type hookInput struct {
	SessionID        string `json:"session_id"`
	CWD              string `json:"cwd"`
	HookEventName    string `json:"hook_event_name"`
	NotificationType string `json:"notification_type"` // Claude Code only
	TranscriptPath   string `json:"transcript_path"`
	Message          string `json:"message"`
}

// tmuxPaneInfo holds pane metadata fetched from tmux.
type tmuxPaneInfo struct {
	PaneID      string
	SessionName string
	WindowIndex int
	PaneIndex   int
	SocketPath  string // tmux server socket path from $TMUX
}

// HandleEvent processes a hook event from any supported agent and updates the pane state file.
func HandleEvent(eventType string, stdin io.Reader, store *state.Store, notifyCfg config.NotifyConfig) error {
	agentDef := resolveAgent()

	var input hookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		// stdin may be empty for some events; treat as empty input
		input = hookInput{}
	}

	paneInfo, err := getTmuxPaneInfo()
	if err != nil {
		// If not running inside tmux, exit silently (don't block the agent)
		fmt.Fprintf(os.Stderr, "tov hook: %v\n", err)
		return nil
	}

	now := time.Now()

	// Check if this event should remove the state file (e.g. SessionEnd for Claude)
	if agentDef.ShouldRemoveOnEvent != nil && agentDef.ShouldRemoveOnEvent(eventType) {
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
	ps.TmuxSocket = paneInfo.SocketPath
	ps.Agent = string(agentDef.Name)

	// Determine new status via agent-specific or default transition
	prevStatus := ps.Status
	var newStatus state.Status
	if agentDef.ApplyTransition != nil {
		newStatus = agentDef.ApplyTransition(eventType, input, ps.Status)
	} else {
		newStatus = defaultStatusTransition(eventType, input, ps.Status)
	}

	message := ""
	if agentDef.StoreMessage != nil {
		message = agentDef.StoreMessage(eventType, input)
	}

	if agentDef.Name == AgentCodex {
		if inferredStatus, inferredMessage := detectCodexState(eventType, paneInfo.PaneID, newStatus); inferredStatus != "" {
			newStatus = inferredStatus
			if inferredMessage != "" {
				message = inferredMessage
			}
		}
	}

	// Update status_changed_at only if status actually changed
	if newStatus != ps.Status {
		ps.Status = newStatus
		ps.StatusChangedAt = now
	}

	// Always update last event
	ps.LastEvent = eventType
	ps.LastEventAt = now

	ps.Message = message

	if err := store.Write(ps); err != nil {
		return err
	}

	// Send macOS notification if enabled and agent supports it
	sentNotification := false
	if notifyCfg.Enabled && agentDef.NotifyTitle != nil {
		title := agentDef.NotifyTitle(eventType)
		if title != "" {
			body := ""
			if agentDef.ExtractNotifyBody != nil {
				body = agentDef.ExtractNotifyBody(eventType, input)
			}
			if body == "" {
				body = input.Message
			}
			sendNotification(title, body, paneInfo, notifyCfg)
			sentNotification = true
		}
	}

	if notifyCfg.Enabled && !sentNotification && prevStatus != newStatus && newStatus == state.StatusWaiting {
		title := fmt.Sprintf("%s - 確認待ち", agentDef.DisplayLabel)
		body := ps.Message
		if body == "" {
			body = input.Message
		}
		if body == "" {
			body = "ユーザーの選択または承認が必要です"
		}
		sendNotification(title, body, paneInfo, notifyCfg)
	}

	return nil
}

// getTmuxPaneInfo reads the current tmux pane information.
func getTmuxPaneInfo() (tmuxPaneInfo, error) {
	paneID := os.Getenv("TMUX_PANE")
	if paneID == "" {
		return tmuxPaneInfo{}, fmt.Errorf("TMUX_PANE not set (not running inside tmux?)")
	}

	// Extract socket path from $TMUX (format: /path/to/socket,pid,session)
	socketPath := ""
	if tmuxEnv := os.Getenv("TMUX"); tmuxEnv != "" {
		if idx := strings.IndexByte(tmuxEnv, ','); idx > 0 {
			socketPath = tmuxEnv[:idx]
		} else {
			socketPath = tmuxEnv
		}
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
		SocketPath:  socketPath,
	}, nil
}

func detectCodexState(eventType, paneID string, fallback state.Status) (state.Status, string) {
	if eventType != "PreToolUse" {
		return "", ""
	}

	for range 3 {
		lines, err := capturePaneContent(paneID, 20)
		if err == nil {
			if waiting, summary := codex.DetectWaiting(lines); waiting {
				return state.StatusWaiting, summary
			}
		}
		time.Sleep(120 * time.Millisecond)
	}

	return fallback, ""
}

func capturePaneContent(paneID string, lines int) ([]string, error) {
	startLine := fmt.Sprintf("-%d", lines)
	out, err := exec.Command("tmux", "capture-pane", "-p", "-e", "-t", paneID, "-S", startLine).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux capture-pane: %w", err)
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}
