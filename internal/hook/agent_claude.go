package hook

import (
	"os"
	"path/filepath"

	"github.com/magcho/tmux-overview/internal/state"
)

func init() {
	registerAgent(claudeAgentDef())
}

func claudeAgentDef() AgentDef {
	return AgentDef{
		Name:         AgentClaude,
		DisplayLabel: "Claude Code",
		HookEvents: []string{
			"SessionStart",
			"UserPromptSubmit",
			"PreToolUse",
			"Notification",
			"Stop",
			"SessionEnd",
		},
		SettingsPath: claudeSettingsPath,
		MatcherValue: "", // empty string
		ApplyTransition: func(eventType string, input hookInput, current state.Status) state.Status {
			return claudeStatusTransition(eventType, input, current)
		},
		ShouldRemoveOnEvent: func(eventType string) bool {
			return eventType == "SessionEnd"
		},
		StoreMessage: func(eventType string, input hookInput) string {
			if eventType == "Notification" {
				return input.NotificationType
			}
			return ""
		},
		NotifyTitle: func(eventType string) string {
			switch eventType {
			case "Notification":
				return "Claude Code - 確認"
			case "Stop":
				return "Claude Code - 完了"
			default:
				return ""
			}
		},
		ExtractNotifyBody: func(eventType string, input hookInput) string {
			switch eventType {
			case "Notification":
				body := extractNotificationContext(input.TranscriptPath)
				if body == "" {
					body = input.Message
				}
				if body == "" {
					body = input.NotificationType
				}
				return body
			case "Stop":
				return extractStopSummary(input.TranscriptPath)
			default:
				return ""
			}
		},
		SetupHooks:  setupClaudeHooks,
		RemoveHooks: removeClaudeHooks,
	}
}

// claudeStatusTransition handles Claude Code-specific status transitions,
// including the Notification event which maps to waiting/done.
func claudeStatusTransition(eventType string, input hookInput, current state.Status) state.Status {
	switch eventType {
	case "Notification":
		switch input.NotificationType {
		case "permission_prompt", "elicitation_dialog":
			return state.StatusWaiting
		case "idle_prompt":
			return state.StatusDone
		default:
			return state.StatusWaiting
		}
	default:
		return defaultStatusTransition(eventType, input, current)
	}
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// setupClaudeHooks adds tov hooks to Claude Code's settings.json.
func setupClaudeHooks(settingsPath, tovCmd string) (map[string]interface{}, error) {
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

	agentDef, _ := GetAgent(AgentClaude)
	for _, event := range agentDef.HookEvents {
		addHookEntry(hooks, event, tovCmd, agentDef.MatcherValue, nil)
	}

	return settings, nil
}

// removeClaudeHooks removes tov hooks from Claude Code's settings.json.
func removeClaudeHooks(settingsPath, tovCmd string) (map[string]interface{}, error) {
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

	agentDef, _ := GetAgent(AgentClaude)
	for _, event := range agentDef.HookEvents {
		removeHookEntry(hooks, event, tovCmd)
	}

	return settings, nil
}
