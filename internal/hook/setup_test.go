package hook

import "testing"

func TestSetupCodexHooksTreatsNullHooksAsEmpty(t *testing.T) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": nil,
					"hooks":   nil,
				},
			},
		},
	}

	hooks := settings["hooks"].(map[string]interface{})
	addHookEntry(hooks, "PreToolUse", "/tmp/tov", nil, nil)

	eventHooks := hooks["PreToolUse"].([]interface{})
	group := eventHooks[0].(map[string]interface{})
	groupHooks := group["hooks"].([]interface{})
	if len(groupHooks) != 1 {
		t.Fatalf("len(groupHooks) = %d, want 1", len(groupHooks))
	}

	hookMap := groupHooks[0].(map[string]interface{})
	if got := hookMap["command"]; got != "/tmp/tov hook PreToolUse" {
		t.Fatalf("command = %v, want %q", got, "/tmp/tov hook PreToolUse")
	}
}

func TestRemoveHookEntryLeavesEmptyHookArray(t *testing.T) {
	hooks := map[string]interface{}{
		"Stop": []interface{}{
			map[string]interface{}{
				"matcher": nil,
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": "/tmp/tov hook Stop",
						"timeout": 5,
					},
				},
			},
		},
	}

	removeHookEntry(hooks, "Stop", "/tmp/tov")

	eventHooks := hooks["Stop"].([]interface{})
	group := eventHooks[0].(map[string]interface{})
	groupHooks, ok := group["hooks"].([]interface{})
	if !ok {
		t.Fatal("group hooks should remain a slice")
	}
	if len(groupHooks) != 0 {
		t.Fatalf("len(groupHooks) = %d, want 0", len(groupHooks))
	}
}

func TestSetupCodexHooksCanReinstallAfterRemove(t *testing.T) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": nil,
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/tmp/tov hook SessionStart",
							"timeout": 5,
						},
					},
				},
			},
		},
	}

	hooks := settings["hooks"].(map[string]interface{})
	removeHookEntry(hooks, "SessionStart", "/tmp/tov")
	addHookEntry(hooks, "SessionStart", "/tmp/tov", nil, nil)

	eventHooks := hooks["SessionStart"].([]interface{})
	group := eventHooks[0].(map[string]interface{})
	groupHooks := group["hooks"].([]interface{})
	if len(groupHooks) != 1 {
		t.Fatalf("len(groupHooks) = %d, want 1", len(groupHooks))
	}

	hookMap := groupHooks[0].(map[string]interface{})
	if got := hookMap["command"]; got != "/tmp/tov hook SessionStart" {
		t.Fatalf("command = %v, want %q", got, "/tmp/tov hook SessionStart")
	}
}
