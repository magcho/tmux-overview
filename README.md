# tov - tmux overseer

tmuxの全セッション・ウィンドウ・ペインを一覧表示し、Claude Codeの実行状態を俯瞰しながら目的のペインへ素早くジャンプできるTUIツール。

```
╭────────────────╮╭──────────────────────────────────────────────╮
│SESSIONS        ││PANE DETAIL                                   │
│▶ All           ││Session   work                                │
│  work          ││Window    1:frontend                          │
│  dev           ││Pane      %21  (active)                       │
│  misc          ││CWD       ~/src/frontend                      │
│                ││Status    🤖 Running  (42s)                   │
│                ││Preview:                                      │
│                ││  ✓ Updated Button.tsx                        │
│                ││  ■ Writing tests...                          │
╰────────────────╯╰──────────────────────────────────────────────╯
╭────────────────────────────────────────────────────────────────╮
│PANE LIST                                                       │
│▶ work:1:frontend   %21 * 🤖 Running (42s)  ~/src/frontend     │
│  work:2:api        %24   💤 Idle           ~/src/api-server   │
│  dev:1:dashboard   %31   🤖 Running (2m)   ~/src/dashboard    │
│  dev:2:infra       %38   ❌ Error          ~/src/infra        │
╰────────────────────────────────────────────────────────────────╯
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

**前提:** Go 1.22以上、tmuxがインストール済みであること。

## 使い方

```bash
# 基本起動
tov

# 特定セッションのみ表示
tov -s work

# 更新間隔を変更（デフォルト: 2秒）
tov --interval 5
```

## キーバインド

| キー | 動作 |
|------|------|
| `↑` / `k` | カーソル上移動 |
| `↓` / `j` | カーソル下移動 |
| `Enter` | 選択ペインへジャンプ |
| `/` | フィルターモード |
| `Esc` | フィルタークリア / セッション選択解除 |
| `r` | 手動リフレッシュ |
| `Space` | プレビュー展開/折畳 |
| `Tab` | セッション一覧とペインリストのフォーカス切替 |
| `1`-`9` | セッション番号で直接絞り込み |
| `q` / `Ctrl+C` | 終了 |

## フィルター

`/` キーでフィルターモードに入り、スペース区切りでAND条件の検索ができます。

```
Filter: biwa running
→ CWDに "biwa" を含み、かつステータスが Running のペインのみ表示
```

検索対象: セッション名、ウィンドウ名、CWD、ステータス、ペイン出力テキスト

## ペインステータス

`tmux capture-pane` の出力を解析して自動判定します。

| ステータス | 判定条件 |
|-----------|---------|
| 🤖 Running | スピナー文字、Analyzing、Writing、Thinking 等を検出 |
| ✅ Done | `> ` プロンプト、`✓ Task completed` を検出 |
| ❌ Error | `Error:`、`Failed to`、`✗` を検出 |
| 💤 Idle | シェルプロンプト (`$`, `%`, `❯`) を検出 |
| ❓ Unknown | いずれにもマッチしない |

## 設定ファイル

`~/.config/tov/config.toml`

```toml
[display]
interval = 2          # 自動更新間隔（秒）
preview_lines = 10    # プレビューの最大行数
cwd_max_length = 40   # CWD表示の最大文字数

[status]
# ステータス判定パターン（正規表現）をカスタマイズ
running_patterns = [
  "[◐◑◒◓⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]",
  "Analyzing",
  "Writing",
  "Thinking",
]

done_patterns = [
  "^> $",
  "✓ Task completed",
]

error_patterns = [
  "Error:",
  "Failed to",
  "✗",
]
```

## ライセンス

MIT
