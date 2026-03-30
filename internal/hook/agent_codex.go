package hook

import (
	"os"
	"path/filepath"
)

func init() {
	registerAgent(codexAgentDef())
}

func codexAgentDef() AgentDef {
	return AgentDef{
		Name:         AgentCodex,
		DisplayLabel: "Codex",
		HookEvents: []string{
			"SessionStart",
			"UserPromptSubmit",
			"PreToolUse",
			"Stop",
		},
		SettingsPath:    codexSettingsPath,
		MatcherValue:    nil, // JSON null
		ApplyTransition: nil, // use defaultStatusTransition
		ShouldRemoveOnEvent: nil, // no SessionEnd; rely on stale cleanup
		StoreMessage:        nil, // no Notification event
		NotifyTitle: func(eventType string) string {
			if eventType == "Stop" {
				return "Codex - 完了"
			}
			return ""
		},
		ExtractNotifyBody: nil, // transcript parsing not yet supported
		SetupHooks:        setupCodexHooks,
		RemoveHooks:       removeCodexHooks,
	}
}

func codexSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "hooks.json"), nil
}

// setupCodexHooks adds tov hooks to Codex's hooks.json.
// Codex uses a standalone hooks.json file (not embedded in a larger settings object).
func setupCodexHooks(settingsPath, tovCmd string) (map[string]interface{}, error) {
	settings, err := readSettings(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	agentDef, _ := GetAgent(AgentCodex)
	for _, event := range agentDef.HookEvents {
		addHookEntry(hooks, event, tovCmd, agentDef.MatcherValue, nil)
	}

	return settings, nil
}

// removeCodexHooks removes tov hooks from Codex's hooks.json.
func removeCodexHooks(settingsPath, tovCmd string) (map[string]interface{}, error) {
	settings, err := readSettings(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return settings, nil
	}

	agentDef, _ := GetAgent(AgentCodex)
	for _, event := range agentDef.HookEvents {
		removeHookEntry(hooks, event, tovCmd)
	}

	return settings, nil
}
