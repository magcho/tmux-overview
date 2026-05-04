# tov - tmux overseer

tmux上で動いているAIコーディングエージェント用ペインを一覧表示し、実行状態を俯瞰しながら目的のペインへ素早くジャンプできるTUIツール。

対応エージェント: **Claude Code**, **Codex**

各エージェントのライフサイクルフックと連携し、状態ファイルが存在するペインだけを対象に、処理中・確認待ち・完了などをリアルタイムで表示します。

```
 tov   │  Agent: 3 panes
╭──────────────────────────────────────────────────────╮
│PANE LIST                                             │
│  DIRECTORY                STATUS            DURATION │
│──────────────────────────────────────────────────────│
│▶ frontend                 🤖 Running        42s     │
│  api-server               ✅ Done           15s     │
│  dashboard                ⏸ Waiting         3s     │
╰──────────────────────────────────────────────────────╯
╭──────────────────────────────────────────────────────╮
│frontend  🤖 Running  (42s)  ~/src/frontend           │
│  > Analyzing component tree...                       │
│  ✓ Updated Button.tsx                                │
│  ✓ Updated index.ts                                  │
│  ■ Writing tests...                                  │
╰──────────────────────────────────────────────────────╯
 [↑↓/jk] 移動  [Enter] ジャンプ  [/] フィルター  [q] 終了
```

## インストール

```bash
# Homebrew（macOS / Linux）
brew install magcho/magcho/tov

# go install
go install github.com/magcho/tmux-overview/cmd/tov@latest

# またはソースからビルド
git clone https://github.com/magcho/tmux-overview.git
cd tmux-overview
make install   # ~/.local/bin/tov にインストール
```

**前提:** tmuxがインストール済みであること。ソースビルドの場合はGo 1.24以上が必要。

## セットアップ

エージェントのフック設定をインストールします。これにより、エージェントが状態変化時に自動で `tov` に通知するようになります。

```bash
# Claude Code のフック設定を追加（デフォルト）
tov setup

# Codex のフック設定を追加
tov setup --agent codex

# 全対応エージェントに一括設定
tov setup --all

# 変更内容をプレビュー（書き込みなし）
tov setup --dry-run
tov setup --agent codex --dry-run

# フック設定を削除
tov setup --remove
tov setup --agent codex --remove
```

| エージェント | 設定ファイル |
|---|---|
| Claude Code | `~/.claude/settings.json` |
| Codex | `~/.codex/hooks.json` |

## 使い方

```bash
# TUI起動
tov

# 更新間隔を変更（デフォルト: 2秒）
tov -interval 5

# ヘルプ表示
tov help

# 終了済みペインのstale状態ファイルを削除
tov cleanup
```

`tov` は tmux の全ペインを無条件に表示するのではなく、Claude Code / Codex の hook 経由で状態ファイルが作られたペインだけを表示します。

## キーバインド

| キー | 動作 |
|------|------|
| `↑` / `k` | カーソル上移動 |
| `↓` / `j` | カーソル下移動 |
| `Enter` | 選択ペインへジャンプ |
| `/` | フィルターモード |
| `Esc` | フィルタークリア |
| `r` | 手動リフレッシュ |
| `q` / `Ctrl+C` | 終了 |

## フィルター

`/` キーでフィルターモードに入り、スペース区切りでAND条件の検索ができます。

```
Filter: biwa running
→ CWDに "biwa" を含み、かつステータスが Running のペインのみ表示
```

検索対象: セッション名、ウィンドウ名、CWD、ステータス、ペイン出力テキスト

## ペインステータス

エージェントのライフサイクルフックを受信してステータスを自動判定します。

| ステータス | 説明 | Claude Code | Codex |
|-----------|------|:-----------:|:-----:|
| 📋 Registered | セッション開始（まだ実行前） | ○ | ○ |
| 🤖 Running | プロンプト送信後、処理中 | ○ | ○ |
| ⏸ Waiting | 権限確認など、ユーザー入力待ち | ○ | △ |
| ✅ Done | タスク完了 | ○ | ○ |

Codex には `Notification` フックがないため、`PreToolUse` 時に tmux pane 末尾を確認し、権限確認やコマンド承認、選択肢付きプロンプトが見えたら `Waiting` に遷移させます。TUI でも同じ判定で表示を補正するため、状態表示と通知が揃います。

## エージェント検出

`tov hook` コマンドは環境変数でエージェントを自動判定します。

| 環境変数 | エージェント |
|---|---|
| `$CODEX_THREAD_ID` が設定されている | Codex |
| それ以外（デフォルト） | Claude Code |

## 通知（macOS）

Notification/Stopイベント時に、macOSデスクトップ通知を送信します（`terminal-notifier` が必要）。Codex では承認や選択待ちを検出したときも通知します。通知をクリックすると該当ペインにジャンプします。

```bash
brew install terminal-notifier
```

## 設定ファイル

`~/.config/tov/config.toml`

```toml
[display]
interval = 2          # 自動更新間隔（秒）
preview_lines = 10    # プレビューの最大行数
cwd_max_length = 40   # CWD表示の最大文字数
language = "en"       # "en" または "ja"

[hook]
# state_dir = "/custom/path"  # 状態ファイルの保存先（デフォルト: $TMPDIR/tov/）

[notify]
enabled = true        # macOS通知の有効/無効
# terminal_app = ""   # ターミナルアプリ名（$TERM_PROGRAMから自動検出）
# sound = ""          # 通知音
# icon = ""           # 通知アイコンのパス
```

補足:

- `display.cwd_max_length` は設定項目として存在しますが、現行UIでは実質未使用です
- `Space` キー入力は実装上受け付けていますが、現行UIでは見た目の変化はありません

## ライセンス

MIT
