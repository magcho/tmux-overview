package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/magcho/tmux-overview/internal/config"
	"github.com/magcho/tmux-overview/internal/hook"
	"github.com/magcho/tmux-overview/internal/state"
	"github.com/magcho/tmux-overview/internal/tmux"
	"github.com/magcho/tmux-overview/internal/tui"
)

func main() {
	// Subcommand routing (before flag.Parse)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("tov %s\n", version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "hook":
			handleHook()
			return
		case "setup":
			handleSetup()
			return
		case "cleanup":
			handleCleanup()
			return
		case "focus":
			handleFocus()
			return
		}
	}

	runTUI()
}

var version = "dev"

func printHelp() {
	fmt.Printf(`tov (tmux overseer) v%s

tmux上のAIコーディングエージェント用ペインを一覧表示し、実行状態を俯瞰できるTUIツール。
対応エージェント: Claude Code, Codex

表示対象:
  Claude Code / Codex のhook経由で状態ファイルが作られたペインのみ表示

Usage:
  tov                       TUI起動
  tov version               バージョン表示
  tov help                  このヘルプを表示
  tov setup [flags]         フック設定をインストール
  tov cleanup               終了済みペインのstale状態ファイルを削除
  tov hook <Event>          フックイベント処理（エージェントから自動呼出し）
  tov focus [flags]         tmuxペインにフォーカス（通知クリック時の内部用）

Setup flags:
  --agent <name>            対象エージェント: claude, codex（デフォルト: claude）
  --all                     全対応エージェントに設定
  --dry-run                 変更をプレビュー（書き込みなし）
  --remove                  tov フック設定を削除

TUI flags:
  -interval <seconds>       自動更新間隔（秒）。config.tomlの設定を上書き

TUI keybindings:
  ↑/k, ↓/j                 カーソル移動
  Enter                     選択ペインにジャンプ
  /                         フィルターモード開始
  Esc                       フィルター解除
  r                         手動更新
  q, Ctrl+C                 終了

Config: ~/.config/tov/config.toml
State:  $TMPDIR/tov/*.json
`, version)
}

func handleHook() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: tov hook <EventType>")
		os.Exit(1)
	}
	eventType := os.Args[2]

	cfg, _ := config.Load()
	store := state.NewStore(cfg.Hook.StateDir)

	if err := hook.HandleEvent(eventType, os.Stdin, store, cfg.Notify); err != nil {
		fmt.Fprintf(os.Stderr, "tov hook: %v\n", err)
		os.Exit(1)
	}
}

func handleSetup() {
	// Parse setup flags
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Preview changes without writing")
	remove := fs.Bool("remove", false, "Remove tov hooks from settings")
	agentFlag := fs.String("agent", "", "Target agent: claude, codex (default: claude)")
	allFlag := fs.Bool("all", false, "Setup hooks for all supported agents")
	fs.Parse(os.Args[2:])

	var agents []hook.AgentName
	switch {
	case *allFlag:
		for _, a := range hook.AllAgents() {
			agents = append(agents, a.Name)
		}
	case *agentFlag != "":
		agents = []hook.AgentName{hook.AgentName(*agentFlag)}
	default:
		agents = []hook.AgentName{hook.AgentClaude}
	}

	if err := hook.Setup(agents, *dryRun, *remove); err != nil {
		fmt.Fprintf(os.Stderr, "tov setup: %v\n", err)
		os.Exit(1)
	}
}

func handleFocus() {
	fs := flag.NewFlagSet("focus", flag.ExitOnError)
	socket := fs.String("socket", "", "tmux socket path")
	target := fs.String("target", "", "tmux target (session:window.pane)")
	app := fs.String("app", "", "terminal application name")
	fs.Parse(os.Args[2:])

	if *socket == "" || *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: tov focus --socket <path> --target <session:window.pane> [--app <name>]")
		os.Exit(1)
	}

	if err := hook.FocusPane(*socket, *target, *app); err != nil {
		fmt.Fprintf(os.Stderr, "tov focus: %v\n", err)
		os.Exit(1)
	}
}

func handleCleanup() {
	cfg, _ := config.Load()
	store := state.NewStore(cfg.Hook.StateDir)

	// Get live tmux panes
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: tmux is not installed or not in PATH")
		os.Exit(1)
	}

	client := tmux.NewClient()
	panes, err := client.ListAllPanes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing panes: %v\n", err)
		os.Exit(1)
	}

	livePaneIDs := make(map[string]bool)
	for _, p := range panes {
		livePaneIDs[p.ID] = true
	}

	removed := store.RemoveStale(livePaneIDs)
	fmt.Printf("Removed %d stale state file(s)\n", removed)
}

func runTUI() {
	// CLI flags
	intervalFlag := flag.Int("interval", 0, "Auto-refresh interval in seconds (overrides config)")
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

	client := tmux.NewClient()
	store := state.NewStore(cfg.Hook.StateDir)
	model := tui.NewModel(client, store, cfg)

	p := tea.NewProgram(model)
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
