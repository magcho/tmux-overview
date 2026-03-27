package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/tmux"
	"github.com/magcho/tmux-overview/internal/tui"
)

func main() {
	// CLI flags
	sessionFlag := flag.String("s", "", "Filter by session name")
	intervalFlag := flag.Int("interval", 0, "Auto-refresh interval in seconds (overrides config)")
	filterFlag := flag.String("filter", "running", "Default filter text (empty string for no filter)")
	flag.Parse()

	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: tmux is not installed or not in PATH")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v (using defaults)\n", err)
		cfg = config.DefaultConfig()
	}

	// Override interval from CLI flag
	if *intervalFlag > 0 {
		cfg.Display.Interval = *intervalFlag
	}

	// Create status detector with config patterns
	var detector *tmux.StatusDetector
	if len(cfg.Status.RunningPatterns) > 0 || len(cfg.Status.DonePatterns) > 0 || len(cfg.Status.ErrorPatterns) > 0 {
		detector = tmux.NewStatusDetectorWithPatterns(
			cfg.Status.RunningPatterns,
			cfg.Status.DonePatterns,
			cfg.Status.ErrorPatterns,
		)
	} else {
		detector = tmux.NewStatusDetector()
	}

	client := tmux.NewClient()
	model := tui.NewModel(client, detector, cfg)

	// Apply session filter from CLI flag
	if *sessionFlag != "" {
		model = model.WithSessionFilter(*sessionFlag)
	}

	// Apply default text filter from CLI flag
	if *filterFlag != "" {
		model = model.WithFilterText(*filterFlag)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// After TUI exits, perform jump if selected
	m := finalModel.(tui.Model)
	if pane := m.JumpPane(); pane != nil {
		if err := client.SwitchToPane(*pane); err != nil {
			fmt.Fprintf(os.Stderr, "Error switching to pane: %v\n", err)
			os.Exit(1)
		}
	}
}
