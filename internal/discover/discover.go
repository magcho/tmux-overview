// Package discover restores missing pane state files by inspecting running
// agents directly, without relying on hook events.
package discover

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/magcho/tmux-overview/internal/state"
)

// Pane is the tmux pane metadata needed to restore a state file.
type Pane struct {
	ID          string
	SessionName string
	WindowIndex int
	PaneIndex   int
	TTY         string // e.g. /dev/ttys001
}

// Process is a process attached to a pane's TTY.
type Process struct {
	PID  int
	Args string
}

// Sources abstracts the system lookups used by Restore.
type Sources struct {
	// ClaudeSessionsDir is Claude Code's per-process session registry (~/.claude/sessions).
	ClaudeSessionsDir string
	TTYProcesses      func(tty string) ([]Process, error)
	IsAlive           func(pid int) bool
	TmuxSocket        string
	Now               func() time.Time
}

// DefaultSources returns Sources backed by the real system.
func DefaultSources() Sources {
	home, _ := os.UserHomeDir()
	return Sources{
		ClaudeSessionsDir: filepath.Join(home, ".claude", "sessions"),
		TTYProcesses:      ttyProcesses,
		IsAlive:           isAlive,
		TmuxSocket:        tmuxSocket(),
		Now:               time.Now,
	}
}

// claudeSession is the subset of ~/.claude/sessions/<pid>.json used here.
type claudeSession struct {
	PID             int    `json:"pid"`
	SessionID       string `json:"sessionId"`
	Kind            string `json:"kind"`
	Entrypoint      string `json:"entrypoint"`
	Status          string `json:"status"`
	Tmux            string `json:"tmux"` // e.g. "20:@37.%55"
	StatusUpdatedAt int64  `json:"statusUpdatedAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

// Restore writes state files for panes that run an agent but have no state file.
// Existing state files are never overwritten. Returns the number of restored panes.
func Restore(store *state.Store, panes []Pane, src Sources) (int, error) {
	existing, err := store.ListAll()
	if err != nil {
		return 0, fmt.Errorf("listing state files: %w", err)
	}
	has := make(map[string]bool, len(existing))
	for _, ps := range existing {
		has[ps.PaneID] = true
	}

	registry := loadClaudeSessions(src)
	now := src.Now()

	restored := 0
	for _, p := range panes {
		if has[p.ID] {
			continue
		}

		ps, ok := fromClaudeSession(registry[p.ID], now)
		if !ok {
			ps, ok = fromTTYProcesses(p.TTY, src, now)
		}
		if !ok {
			continue
		}

		ps.PaneID = p.ID
		ps.SessionName = p.SessionName
		ps.WindowIndex = p.WindowIndex
		ps.PaneIndex = p.PaneIndex
		ps.TmuxSocket = src.TmuxSocket
		ps.LastEvent = "Restore"
		ps.LastEventAt = now

		if err := store.Write(ps); err != nil {
			return restored, err
		}
		restored++
	}
	return restored, nil
}

// loadClaudeSessions maps pane IDs to the newest live interactive Claude session.
func loadClaudeSessions(src Sources) map[string]*claudeSession {
	result := make(map[string]*claudeSession)
	entries, err := os.ReadDir(src.ClaudeSessionsDir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src.ClaudeSessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s claudeSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.Tmux == "" || s.Kind != "interactive" || (s.Entrypoint != "" && s.Entrypoint != "cli") {
			continue
		}
		if !src.IsAlive(s.PID) {
			continue
		}
		idx := strings.LastIndexByte(s.Tmux, '%')
		if idx < 0 {
			continue
		}
		paneID := s.Tmux[idx:]
		if cur, ok := result[paneID]; ok && cur.UpdatedAt >= s.UpdatedAt {
			continue
		}
		result[paneID] = &s
	}
	return result
}

func fromClaudeSession(s *claudeSession, now time.Time) (state.PaneState, bool) {
	if s == nil {
		return state.PaneState{}, false
	}
	changedAt := now
	if s.StatusUpdatedAt > 0 {
		changedAt = time.UnixMilli(s.StatusUpdatedAt)
	}
	return state.PaneState{
		Agent:           "claude",
		SessionID:       s.SessionID,
		PID:             s.PID,
		Status:          mapClaudeStatus(s.Status),
		StatusChangedAt: changedAt,
	}, true
}

func mapClaudeStatus(s string) state.Status {
	switch s {
	case "idle":
		return state.StatusDone
	case "busy":
		return state.StatusRunning
	case "waiting":
		return state.StatusWaiting
	default:
		return state.StatusRegistered
	}
}

// fromTTYProcesses detects an interactive agent process on the pane's TTY.
// Used for agents that do not keep a session registry (older Claude Code, Codex).
func fromTTYProcesses(tty string, src Sources, now time.Time) (state.PaneState, bool) {
	if tty == "" || src.TTYProcesses == nil {
		return state.PaneState{}, false
	}
	procs, err := src.TTYProcesses(tty)
	if err != nil {
		return state.PaneState{}, false
	}
	for _, proc := range procs {
		agent := interactiveAgent(proc.Args)
		if agent == "" {
			continue
		}
		return state.PaneState{
			Agent:           agent,
			PID:             proc.PID,
			Status:          state.StatusRegistered,
			StatusChangedAt: now,
		}, true
	}
	return state.PaneState{}, false
}

// interactiveAgent returns the agent name if args is an interactive claude/codex invocation.
func interactiveAgent(args string) string {
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return ""
	}
	agent := filepath.Base(fields[0])
	switch agent {
	case "claude":
		for _, f := range fields[1:] {
			if f == "-p" || f == "--print" {
				return ""
			}
		}
	case "codex":
		if len(fields) > 1 && fields[1] == "exec" {
			return ""
		}
	default:
		return ""
	}
	return agent
}

func ttyProcesses(tty string) ([]Process, error) {
	out, err := exec.Command("ps", "-t", strings.TrimPrefix(tty, "/dev/"), "-o", "pid=,args=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps -t %s: %w", tty, err)
	}
	var procs []Process
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		pidStr, args, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		procs = append(procs, Process{PID: pid, Args: strings.TrimSpace(args)})
	}
	return procs, nil
}

func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// tmuxSocket extracts the server socket path from $TMUX (format: /path/to/socket,pid,session).
func tmuxSocket() string {
	socket, _, _ := strings.Cut(os.Getenv("TMUX"), ",")
	return socket
}
