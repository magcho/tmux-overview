package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client is the interface for interacting with tmux.
type Client interface {
	ListSessions() ([]Session, error)
	ListAllPanes() ([]Pane, error)
	CapturePaneContent(paneID string, lines int) ([]string, error)
	SwitchToPane(pane Pane) error
	IsInsideTmux() bool
}

type client struct{}

// NewClient creates a new tmux client.
func NewClient() Client {
	return &client{}
}

func (c *client) IsInsideTmux() bool {
	return exec.Command("tmux", "display-message", "-p", "#{session_name}").Run() == nil
}

func runTmux(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func (c *client) ListSessions() ([]Session, error) {
	out, err := runTmux("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	var sessions []Session
	for _, name := range strings.Split(out, "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		sessions = append(sessions, Session{Name: name})
	}
	return sessions, nil
}

func (c *client) ListAllPanes() ([]Pane, error) {
	// Format: session_name, window_index, window_name, window_active, pane_id, pane_index, pane_current_path, pane_pid, pane_width, pane_height, pane_active
	format := "#{session_name}\t#{window_index}\t#{window_name}\t#{window_active}\t#{pane_id}\t#{pane_index}\t#{pane_current_path}\t#{pane_pid}\t#{pane_width}\t#{pane_height}\t#{pane_active}"
	out, err := runTmux("list-panes", "-a", "-F", format)
	if err != nil {
		return nil, fmt.Errorf("listing panes: %w", err)
	}

	var panes []Pane
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 11 {
			continue
		}

		windowIndex, _ := strconv.Atoi(fields[1])
		windowActive := fields[3] == "1"
		paneIndex, _ := strconv.Atoi(fields[5])
		pid, _ := strconv.Atoi(fields[7])
		width, _ := strconv.Atoi(fields[8])
		height, _ := strconv.Atoi(fields[9])
		active := fields[10] == "1"

		p := Pane{
			SessionName:  fields[0],
			WindowIndex:  windowIndex,
			WindowName:   fields[2],
			WindowActive: windowActive,
			ID:           fields[4],
			Index:        paneIndex,
			CWD:          fields[6],
			PID:          pid,
			Width:        width,
			Height:       height,
			Active:       active,
		}
		panes = append(panes, p)
	}

	return panes, nil
}

func (c *client) CapturePaneContent(paneID string, lines int) ([]string, error) {
	startLine := fmt.Sprintf("-%d", lines)
	out, err := runTmux("capture-pane", "-p", "-e", "-t", paneID, "-S", startLine)
	if err != nil {
		return nil, fmt.Errorf("capturing pane %s: %w", paneID, err)
	}
	return strings.Split(out, "\n"), nil
}

func (c *client) SwitchToPane(pane Pane) error {
	target := fmt.Sprintf("%s:%d", pane.SessionName, pane.WindowIndex)

	if c.IsInsideTmux() {
		if _, err := runTmux("select-window", "-t", target); err != nil {
			return fmt.Errorf("selecting window: %w", err)
		}
		if _, err := runTmux("select-pane", "-t", pane.ID); err != nil {
			return fmt.Errorf("selecting pane: %w", err)
		}
		// Switch to the session if it's different from the current one
		if _, err := runTmux("switch-client", "-t", pane.SessionName); err != nil {
			return fmt.Errorf("switching client: %w", err)
		}
	} else {
		_, err := runTmux("attach-session", "-t", target)
		if err != nil {
			return fmt.Errorf("attaching session: %w", err)
		}
	}
	return nil
}
