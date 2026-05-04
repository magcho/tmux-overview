package hook

import (
	"os"

	"github.com/magcho/tmux-overview/internal/state"
)

// AgentName identifies a supported coding agent.
type AgentName string

const (
	AgentClaude AgentName = "claude"
	AgentCodex  AgentName = "codex"
)

// AgentDef describes the hook behavior for a specific coding agent.
// Function fields that are nil fall back to default behavior.
type AgentDef struct {
	Name         AgentName
	DisplayLabel string   // e.g. "Claude Code", "Codex"
	HookEvents   []string // Hook events this agent supports

	// SettingsPath returns the path to the agent's hook config file.
	SettingsPath func() (string, error)
	// MatcherValue is the JSON value for the "matcher" field ("" for Claude, nil for Codex).
	MatcherValue interface{}
	// ExtraHookFields are additional fields merged into each hook entry (e.g. timeout overrides).
	ExtraHookFields map[string]interface{}

	// ApplyTransition maps (eventType, hookInput, currentStatus) -> newStatus.
	// If nil, defaultStatusTransition is used.
	ApplyTransition func(eventType string, input hookInput, current state.Status) state.Status
	// ShouldRemoveOnEvent returns true if the state file should be deleted for this event.
	// If nil, state files are never deleted by event (rely on stale cleanup).
	ShouldRemoveOnEvent func(eventType string) bool
	// StoreMessage extracts a message to store in PaneState.Message.
	// If nil, Message is always cleared.
	StoreMessage func(eventType string, input hookInput) string
	// NotifyTitle returns the notification title for a given event. Empty string means no notification.
	// If nil, notifications are disabled for this agent.
	NotifyTitle func(eventType string) string
	// ExtractNotifyBody extracts notification body text.
	// If nil, falls back to input.Message.
	ExtractNotifyBody func(eventType string, input hookInput) string

	// SetupHooks writes hook config for this agent. Returns the resulting settings map.
	SetupHooks func(settingsPath, tovCmd string) (map[string]interface{}, error)
	// RemoveHooks removes tov hooks from this agent's config. Returns the resulting settings map.
	RemoveHooks func(settingsPath, tovCmd string) (map[string]interface{}, error)
}

var agentRegistry = map[AgentName]AgentDef{}

func registerAgent(def AgentDef) {
	agentRegistry[def.Name] = def
}

// GetAgent returns the AgentDef for the given name.
func GetAgent(name AgentName) (AgentDef, bool) {
	a, ok := agentRegistry[name]
	return a, ok
}

// AllAgents returns all registered agent definitions.
func AllAgents() []AgentDef {
	defs := make([]AgentDef, 0, len(agentRegistry))
	for _, d := range agentRegistry {
		defs = append(defs, d)
	}
	return defs
}

// DetectAgent determines the calling agent from environment variables.
// Returns AgentClaude as the default (backward compatible).
func DetectAgent() AgentName {
	if isCodexEnvironment() {
		return AgentCodex
	}
	return AgentClaude
}

func isCodexEnvironment() bool {
	for _, key := range []string{
		"CODEX_THREAD_ID",
		"CODEX_HOME",
		"CODEX_SANDBOX",
		"CODEX_SANDBOX_NETWORK_DISABLED",
		"CODEX_CI",
	} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

// resolveAgent detects the agent and returns its definition, falling back to Claude.
func resolveAgent() AgentDef {
	name := DetectAgent()
	if def, ok := GetAgent(name); ok {
		return def
	}
	def, _ := GetAgent(AgentClaude)
	return def
}

// defaultStatusTransition handles the shared event-to-status mapping
// common to all agents.
func defaultStatusTransition(eventType string, _ hookInput, current state.Status) state.Status {
	switch eventType {
	case "SessionStart":
		return state.StatusRegistered
	case "UserPromptSubmit":
		return state.StatusRunning
	case "PreToolUse":
		return state.StatusRunning
	case "Stop":
		return state.StatusDone
	default:
		return current
	}
}
