package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Display DisplayConfig `toml:"display"`
	Hook    HookConfig    `toml:"hook"`
	Notify  NotifyConfig  `toml:"notify"`
}

type DisplayConfig struct {
	Interval     int    `toml:"interval"`
	PreviewLines int    `toml:"preview_lines"`
	CWDMaxLength int    `toml:"cwd_max_length"`
	Language     string `toml:"language"`
}

type HookConfig struct {
	StateDir string `toml:"state_dir"` // Override state directory (default: $TMPDIR/tov/)
}

type NotifyConfig struct {
	Enabled     bool   `toml:"enabled"`      // Send macOS notifications (default: true)
	TerminalApp string `toml:"terminal_app"` // Terminal app name (auto-detect from $TERM_PROGRAM if empty)
	Sound       string `toml:"sound"`        // terminal-notifier -sound argument
	Icon        string `toml:"icon"`         // Notification icon path
}

func DefaultConfig() Config {
	return Config{
		Display: DisplayConfig{
			Interval:     2,
			PreviewLines: 10,
			CWDMaxLength: 40,
			Language:     "en",
		},
		Notify: NotifyConfig{
			Enabled: true,
		},
	}
}

func Load() (Config, error) {
	cfg := DefaultConfig()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // use defaults
	}

	configPath := filepath.Join(homeDir, ".config", "tov", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // no config file, use defaults
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", configPath, err)
	}

	// Apply defaults for unset values
	if cfg.Display.Interval <= 0 {
		cfg.Display.Interval = 2
	}
	if cfg.Display.PreviewLines <= 0 {
		cfg.Display.PreviewLines = 10
	}
	if cfg.Display.CWDMaxLength <= 0 {
		cfg.Display.CWDMaxLength = 40
	}
	if cfg.Display.Language == "" {
		cfg.Display.Language = "en"
	}

	return cfg, nil
}
