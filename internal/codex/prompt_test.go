package codex

import "testing"

func TestDetectWaitingPermissionPrompt(t *testing.T) {
	preview := []string{
		"• Calling",
		"  └ vibe_kanban.create_issue_relationship(...)",
		"Field 1/1",
		"Allow the vibe_kanban MCP server to run tool \"create_issue_relationship\"?",
		"enter to submit | esc to cancel",
	}

	waiting, summary := DetectWaiting(preview)
	if !waiting {
		t.Fatal("DetectWaiting() = false, want true")
	}
	if summary == "" {
		t.Fatal("DetectWaiting() summary = empty, want non-empty")
	}
}

func TestDetectWaitingCommandApprovalPrompt(t *testing.T) {
	preview := []string{
		"Would you like to run the following command?",
		"Reason: Do you want to let me inspect the exact changed files in Renovate PR #7 to compare it with the other dependency PRs?",
		"$ gh pr diff 7 --repo magcho/tmux-overview --name-only",
		"1. Yes, proceed (y)",
		"2. Yes, and don't ask again for commands that start with `gh pr diff` (p)",
		"3. No, and tell Codex what to do differently (esc)",
		"Press enter to confirm or esc to cancel",
	}

	waiting, summary := DetectWaiting(preview)
	if !waiting {
		t.Fatal("DetectWaiting() = false, want true")
	}
	if summary != "$ gh pr diff 7 --repo magcho/tmux-overview --name-only" {
		t.Fatalf("DetectWaiting() summary = %q, want command line", summary)
	}
}

func TestDetectWaitingHookTrailer(t *testing.T) {
	preview := []string{
		"• Running Stop hook",
		"",
		"Stop hook (completed)",
	}

	waiting, summary := DetectWaiting(preview)
	if !waiting {
		t.Fatal("DetectWaiting() = false, want true")
	}
	if summary != "Stop hook (completed)" {
		t.Fatalf("DetectWaiting() summary = %q, want %q", summary, "Stop hook (completed)")
	}
}

func TestDetectWaitingIgnoresNormalOutput(t *testing.T) {
	preview := []string{
		"PR #7 と #8 はどちらも同じ手修正コミットが追加されていて、実質的に同じ v2 移行を別 PR に積んでいる状態です。",
		"main との差分と現在の依存定義を見て、どちらを残すべきかを詰めます。",
	}

	waiting, summary := DetectWaiting(preview)
	if waiting {
		t.Fatalf("DetectWaiting() = true, want false; summary=%q", summary)
	}
}
