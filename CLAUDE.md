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

## Architecture

bubbletea の Elm Architecture (Model → Update → View) に従った構成:

- **`cmd/tov/main.go`** — エントリポイント。CLIフラグ解析、config読込、tmux Client / StatusDetector 生成、bubbletea Program 起動、終了後のペインジャンプ実行
- **`internal/tmux/`** — tmuxとのやり取りを `Client` インターフェースで抽象化。他パッケージは必ずインターフェース経由で利用する
  - `client.go` — `tmux` コマンドのサブプロセス呼び出し（`list-panes -a`, `capture-pane -p` 等）
  - `session.go` — Session / Window / Pane 構造体、PaneStatus enum
  - `status.go` — `capture-pane` 出力から Running/Done/Error/Idle を正規表現で判定する StatusDetector
- **`internal/tui/`** — bubbletea の TUI レイヤー
  - `model.go` — Model定義、Init、データ取得コマンド、visiblePanes()によるフィルタ適用
  - `update.go` — キー入力・tick・ペイン更新メッセージのハンドリング
  - `view.go` — lipgloss によるレイアウト描画。全スタイル定義はこのファイル先頭に集約
  - `filter.go` — スペース区切りAND条件のインクリメンタルフィルタ
  - `keys.go` — keyAction enum とキーマッピング
- **`internal/config/`** — `~/.config/tov/config.toml` の読み込み（TOML形式、BurntSushi/toml使用）

## Key Design Decisions

- tmuxコマンド呼び出しは全て `internal/tmux/client.go` の `runTmux()` に集約
- StatusDetector のパターンはデフォルト定数を持ちつつ、config.toml でオーバーライド可能
- `tea.Tick` による2秒間隔の自動更新（configurable）
- ペインジャンプはTUI終了後に `main.go` で実行（TUI中にtmuxセッション切替するとターミナルが壊れるため）
- エラーは `fmt.Errorf` でラップして上位に伝播（`log.Fatal` は main 以外では使わない）
- View のレイアウト計算はターミナル高に基づく動的分配（middleInner / listInner）
