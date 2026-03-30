package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Setup adds or removes tov hooks for the specified agents.
func Setup(agents []AgentName, dryRun bool, remove bool) error {
	tovCmd, err := tovBinaryPath()
	if err != nil {
		return err
	}

	for _, agentName := range agents {
		agentDef, ok := GetAgent(agentName)
		if !ok {
			return fmt.Errorf("unknown agent: %s", agentName)
		}

		settingsPath, err := agentDef.SettingsPath()
		if err != nil {
			return fmt.Errorf("getting settings path for %s: %w", agentDef.DisplayLabel, err)
		}

		var settings map[string]interface{}
		if remove {
			if agentDef.RemoveHooks == nil {
				continue
			}
			settings, err = agentDef.RemoveHooks(settingsPath, tovCmd)
			if err != nil {
				return fmt.Errorf("removing hooks for %s: %w", agentDef.DisplayLabel, err)
			}
			fmt.Printf("Removing tov hooks from %s (%s)\n", agentDef.DisplayLabel, settingsPath)
		} else {
			if agentDef.SetupHooks == nil {
				continue
			}
			settings, err = agentDef.SetupHooks(settingsPath, tovCmd)
			if err != nil {
				return fmt.Errorf("adding hooks for %s: %w", agentDef.DisplayLabel, err)
			}
			fmt.Printf("Adding tov hooks to %s (%s)\n", agentDef.DisplayLabel, settingsPath)
		}

		if dryRun {
			out, _ := json.MarshalIndent(settings, "", "  ")
			fmt.Printf("\n--- Preview (%s) ---\n", settingsPath)
			fmt.Println(string(out))
			continue
		}

		if err := writeSettings(settingsPath, settings); err != nil {
			return fmt.Errorf("writing settings for %s: %w", agentDef.DisplayLabel, err)
		}
	}

	return nil
}

// === Shared utilities for setup ===

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

// addHookEntry adds a tov hook entry to a hooks map for a given event.
// matcherValue is the JSON value for the "matcher" field ("" for Claude, nil for Codex).
func addHookEntry(hooks map[string]interface{}, event, tovCmd string, matcherValue interface{}, extraFields map[string]interface{}) {
	command := fmt.Sprintf("%s hook %s", tovCmd, event)
	newHook := map[string]interface{}{
		"type":    "command",
		"command": command,
		"timeout": 5,
	}
	for k, v := range extraFields {
		newHook[k] = v
	}

	eventHooks, ok := hooks[event].([]interface{})
	if !ok {
		// No existing hooks for this event; create new entry
		hooks[event] = []interface{}{
			map[string]interface{}{
				"matcher": matcherValue,
				"hooks":   []interface{}{newHook},
			},
		}
		return
	}

	// Check if tov hook already exists in any matcher group
	if hasTovHook(eventHooks, tovCmd) {
		return
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
			"matcher": matcherValue,
			"hooks":   []interface{}{newHook},
		})
	}
}

// removeHookEntry removes tov hook entries from a hooks map for a given event.
func removeHookEntry(hooks map[string]interface{}, event, tovCmd string) {
	eventHooks, ok := hooks[event].([]interface{})
	if !ok {
		return
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
	return strings.Contains(cmd, tovCmd+" hook ") || strings.HasSuffix(cmd, tovCmd+" hook")
}
