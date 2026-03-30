# tov - tmux overseer

tmuxの全セッション・ウィンドウ・ペインを一覧表示し、Claude Codeの実行状態を俯瞰しながら目的のペインへ素早くジャンプできるTUIツール。

Claude Codeのライフサイクルフックと連携し、各ペインの状態（処理中・確認待ち・完了など）をリアルタイムで表示します。

```
 tov   │  Claude: 3 panes
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
# go install
go install github.com/magcho/tmux-overview/cmd/tov@latest

# またはソースからビルド
git clone https://github.com/magcho/tmux-overview.git
cd tmux-overview
make install   # ~/.local/bin/tov にインストール
```

**前提:** Go 1.24以上、tmuxがインストール済みであること。

## セットアップ

Claude Codeのフック設定をインストールします。これにより、Claude Codeが状態変化時に自動で `tov` に通知するようになります。

```bash
# フック設定を ~/.claude/settings.json に追加
tov setup

# 変更内容をプレビュー（書き込みなし）
tov setup --dry-run

# フック設定を削除
tov setup --remove
```

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

## キーバインド

| キー | 動作 |
|------|------|
| `↑` / `k` | カーソル上移動 |
| `↓` / `j` | カーソル下移動 |
| `Enter` | 選択ペインへジャンプ |
| `/` | フィルターモード |
| `Esc` | フィルタークリア |
| `r` | 手動リフレッシュ |
| `Space` | プレビュー展開/折畳 |
| `q` / `Ctrl+C` | 終了 |

## フィルター

`/` キーでフィルターモードに入り、スペース区切りでAND条件の検索ができます。

```
Filter: biwa running
→ CWDに "biwa" を含み、かつステータスが Running のペインのみ表示
```

検索対象: セッション名、ウィンドウ名、CWD、ステータス、ペイン出力テキスト

## ペインステータス

Claude Codeのライフサイクルフック（SessionStart, UserPromptSubmit, PreToolUse, Notification, Stop, SessionEnd）を受信して自動判定します。

| ステータス | 説明 |
|-----------|------|
| 📋 Registered | Claude Codeセッション開始（まだ実行前） |
| 🤖 Running | プロンプト送信後、処理中 |
| ⏸ Waiting | 権限確認など、ユーザー入力待ち |
| ✅ Done | タスク完了 |

## 通知（macOS）

Notification/Stopイベント時に、macOSデスクトップ通知を送信します（`terminal-notifier` が必要）。通知をクリックすると該当ペインにジャンプします。

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

## ライセンス

MIT
