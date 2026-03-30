package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// tovHookEvents lists the Claude Code hook events that tov needs.
var tovHookEvents = []string{
	"SessionStart",
	"UserPromptSubmit",
	"PreToolUse",
	"Notification",
	"Stop",
	"SessionEnd",
}

// Setup adds or removes tov hooks from ~/.claude/settings.json.
func Setup(dryRun bool, remove bool) error {
	settingsPath, err := claudeSettingsPath()
	if err != nil {
		return err
	}

	settings, err := readSettings(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading settings: %w", err)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	tovCmd, err := tovBinaryPath()
	if err != nil {
		return err
	}

	if remove {
		removeTovHooks(settings, tovCmd)
		fmt.Println("Removing tov hooks from", settingsPath)
	} else {
		addTovHooks(settings, tovCmd)
		fmt.Println("Adding tov hooks to", settingsPath)
	}

	if dryRun {
		out, _ := json.MarshalIndent(settings, "", "  ")
		fmt.Println("\n--- Preview ---")
		fmt.Println(string(out))
		return nil
	}

	return writeSettings(settingsPath, settings)
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func readSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]interface{}) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating settings dir: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}
	return nil
}

func tovBinaryPath() (string, error) {
	// Try to find tov in PATH first
	path, err := exec.LookPath("tov")
	if err == nil {
		return path, nil
	}
	// Fall back to the currently running binary
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine tov binary path: %w", err)
	}
	return exe, nil
}

func addTovHooks(settings map[string]interface{}, tovCmd string) {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	for _, event := range tovHookEvents {
		command := fmt.Sprintf("%s hook %s", tovCmd, event)
		newHook := map[string]interface{}{
			"type":    "command",
			"command": command,
			"timeout": 5,
		}

		eventHooks, ok := hooks[event].([]interface{})
		if !ok {
			// No existing hooks for this event; create new entry
			hooks[event] = []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks":   []interface{}{newHook},
				},
			}
			continue
		}

		// Check if tov hook already exists in any matcher group
		if hasTovHook(eventHooks, tovCmd) {
			continue
		}

		// Append tov hook to the first matcher group, or create a new one
		if len(eventHooks) > 0 {
			if group, ok := eventHooks[0].(map[string]interface{}); ok {
				if groupHooks, ok := group["hooks"].([]interface{}); ok {
					group["hooks"] = append(groupHooks, newHook)
				}
			}
		} else {
			hooks[event] = append(eventHooks, map[string]interface{}{
				"matcher": "",
				"hooks":   []interface{}{newHook},
			})
		}
	}
}

func removeTovHooks(settings map[string]interface{}, tovCmd string) {
	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return
	}

	for _, event := range tovHookEvents {
		eventHooks, ok := hooks[event].([]interface{})
		if !ok {
			continue
		}

		for _, matcherGroup := range eventHooks {
			group, ok := matcherGroup.(map[string]interface{})
			if !ok {
				continue
			}
			groupHooks, ok := group["hooks"].([]interface{})
			if !ok {
				continue
			}

			var filtered []interface{}
			for _, h := range groupHooks {
				hookMap, ok := h.(map[string]interface{})
				if !ok {
					filtered = append(filtered, h)
					continue
				}
				cmd, _ := hookMap["command"].(string)
				if !isTovCommand(cmd, tovCmd) {
					filtered = append(filtered, h)
				}
			}
			group["hooks"] = filtered
		}
	}
}

func hasTovHook(eventHooks []interface{}, tovCmd string) bool {
	for _, matcherGroup := range eventHooks {
		group, ok := matcherGroup.(map[string]interface{})
		if !ok {
			continue
		}
		groupHooks, ok := group["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range groupHooks {
			hookMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hookMap["command"].(string)
			if isTovCommand(cmd, tovCmd) {
				return true
			}
		}
	}
	return false
}

func isTovCommand(cmd string, tovCmd string) bool {
	return len(cmd) >= 8 && (cmd == tovCmd+" hook SessionStart" ||
		cmd == tovCmd+" hook UserPromptSubmit" ||
		cmd == tovCmd+" hook PreToolUse" ||
		cmd == tovCmd+" hook Notification" ||
		cmd == tovCmd+" hook Stop" ||
		cmd == tovCmd+" hook SessionEnd")
}
