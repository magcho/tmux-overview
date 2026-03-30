# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tov** (tmux overseer) — tmuxの全セッション・ウィンドウ・ペインを一覧表示し、Claude Codeの実行状態を俯瞰しながら目的のペインへジャンプできるTUIツール。Go + bubbletea + lipgloss で構築。

## Build & Run

```bash
make build          # ./tov にバイナリ生成
make install        # ~/.local/bin/tov にインストール
make clean          # バイナリ削除
go test ./...       # 全テスト実行
go test ./internal/tui/ -run TestViewHeight24  # 単体テスト実行例
```

前提: Go 1.24+, tmux がPATHに存在すること。

### サブコマンド

```bash
tov                 # TUI起動（デフォルト）
tov hook <Event>    # Claude Codeフックイベント処理（フックから自動呼出し）
tov setup           # Claude Code settings.jsonにフック設定を追加
tov setup --dry-run # 変更プレビュー
tov setup --remove  # フック設定を削除
tov cleanup         # 終了済みペインのstale状態ファイルを削除
```

## Architecture

bubbletea の Elm Architecture (Model → Update → View) に従った構成:

- **`cmd/tov/main.go`** — エントリポイント。サブコマンドルーティング（hook/setup/cleanup）、TUI起動、終了後のペインジャンプ実行
- **`internal/state/`** — ペインごとのJSON状態ファイル管理（`$TMPDIR/tov/`）
  - `types.go` — PaneState構造体、Status定数（registered/running/waiting/done）
  - `store.go` — 状態ファイルのアトミック書き込み・読み込み・一覧・削除・staleクリーンアップ
- **`internal/hook/`** — Claude Codeライフサイクルフックの処理
  - `handler.go` — フックイベント受信 → ステータス遷移 → 状態ファイル更新
  - `setup.go` — `~/.claude/settings.json` へのフック設定自動追加/削除
- **`internal/tmux/`** — tmuxとのやり取りを `Client` インターフェースで抽象化
  - `client.go` — `tmux` コマンドのサブプロセス呼び出し（`list-panes -a`, `capture-pane -p` 等）
  - `session.go` — Session / Window / Pane 構造体、PaneStatus enum
- **`internal/tui/`** — bubbletea の TUI レイヤー
  - `model.go` — Model定義、Init、state.Storeからのデータ取得、visiblePanes()によるフィルタ適用
  - `update.go` — キー入力・tick・ペイン更新メッセージのハンドリング
  - `view.go` — lipgloss によるレイアウト描画。全スタイル定義はこのファイル先頭に集約
  - `filter.go` — スペース区切りAND条件のインクリメンタルフィルタ
  - `keys.go` — keyAction enum とキーマッピング
- **`internal/config/`** — `~/.config/tov/config.toml` の読み込み（TOML形式、BurntSushi/toml使用）

## Key Design Decisions

- **ステータス検出はClaude Codeフックベース**: capture-pane正規表現ではなく、Claude Codeのライフサイクルフック（SessionStart, UserPromptSubmit, PreToolUse, Notification, Stop, SessionEnd）でステータスを取得
- **状態管理はペインごとのJSONファイル**: `$TMPDIR/tov/%ID.json` にアトミック書き込み（temp+rename）。ファイルロック不要
- tmuxコマンド呼び出しは全て `internal/tmux/client.go` の `runTmux()` に集約
- `capture-pane` はプレビュー表示用にのみ使用（ステータス検出には使わない）
- `tea.Tick` による2秒間隔の自動更新（configurable）
- ペインジャンプはTUI終了後に `main.go` で実行（TUI中にtmuxセッション切替するとターミナルが壊れるため）
- エラーは `fmt.Errorf` でラップして上位に伝播（`log.Fatal` は main 以外では使わない）
- View のレイアウト計算はターミナル高に基づく動的分配（listInner / previewInner）
- stale状態ファイルはTUIポーリング時に自動クリーンアップ（tmuxペイン存在チェック）
