package hook

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// FocusPane activates the terminal app and switches the most active tmux client
// to the specified pane. Called when a notification is clicked.
func FocusPane(socketPath, target, app string) error {
	// Activate terminal application
	if app != "" {
		exec.Command("open", "-a", app).Run()
	}

	// Find the most recently active tmux client
	clientTTY, err := findActiveClient(socketPath)
	if err != nil {
		return fmt.Errorf("finding active client: %w", err)
	}

	// Switch client to the target pane
	if _, err := runTmuxWithSocket(socketPath, "switch-client", "-c", clientTTY, "-t", target); err != nil {
		return fmt.Errorf("switching to %s: %w", target, err)
	}

	return nil
}

// findActiveClient returns the TTY of the most recently active tmux client.
func findActiveClient(socketPath string) (string, error) {
	out, err := runTmuxWithSocket(socketPath, "list-clients", "-F", "#{client_activity} #{client_name}")
	if err != nil {
		return "", fmt.Errorf("listing clients: %w", err)
	}

	type clientEntry struct {
		activity int
		name     string
	}

	var clients []clientEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		activity, _ := strconv.Atoi(parts[0])
		clients = append(clients, clientEntry{activity: activity, name: parts[1]})
	}

	if len(clients) == 0 {
		return "", fmt.Errorf("no tmux clients found")
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].activity > clients[j].activity
	})

	return clients[0].name, nil
}

// runTmuxWithSocket executes a tmux command with an explicit socket path.
func runTmuxWithSocket(socketPath string, args ...string) (string, error) {
	fullArgs := append([]string{"-S", socketPath}, args...)
	cmd := exec.Command("tmux", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux -S %s %s: %w", socketPath, strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
