# tov 仕様書

## 概要

`tov` は、tmux 上で動いている AI コーディングエージェントの状態を一覧し、対象ペインへ素早くジャンプするための Go 製 TUI ツールである。

現行実装は tmux の全ペインを常時監視する方式ではなく、**Claude Code / Codex のフック経由で状態ファイルが作られたペインだけ**を対象に表示する。

コマンド名: `tov`（tmux overseer）

---

## 目的

- 複数の tmux ペインで走っている AI エージェントの進行状況を俯瞰する
- 確認待ちや完了したペインをすぐ見つける
- 通知や一覧画面から目的のペインへ即座にフォーカスする

---

## 技術スタック

| 項目 | 選定 |
|------|------|
| 言語 | Go |
| TUI | Bubble Tea + Lip Gloss |
| tmux 連携 | `tmux` コマンド呼び出し |
| 状態管理 | pane ごとの JSON ファイル |
| 対応エージェント | Claude Code, Codex |

---

## アーキテクチャ

### 1. フックで状態を収集

各エージェントの hook から `tov hook <Event>` が呼ばれ、現在の tmux pane 情報を取得して状態ファイルを更新する。

- Claude Code: `~/.claude/settings.json`
- Codex: `~/.codex/hooks.json`

状態ファイルの保存先:

- デフォルト: `$TMPDIR/tov/`
- 設定で上書き可能: `[hook].state_dir`

### 2. TUI は状態ファイルのある pane だけ表示

TUI 起動時と定期更新時に以下を行う。

1. `tmux list-panes -a` で全 pane を取得
2. 状態ファイルを読み込む
3. 状態ファイルが存在する pane だけを agent pane として表示対象にする
4. `tmux capture-pane` で末尾プレビューを取得する
5. 現在存在しない pane の stale 状態ファイルを削除する

---

## 起動とサブコマンド

```bash
# TUI 起動
tov

# バージョン表示
tov version

# ヘルプ表示
tov help

# hook 設定の追加
tov setup
tov setup --agent codex
tov setup --all

# hook 設定の削除
tov setup --remove

# 変更内容のプレビュー
tov setup --dry-run

# stale 状態ファイルの削除
tov cleanup

# 通知クリック時の内部用フォーカス
tov focus --socket <path> --target <session:window.pane>
```

### TUI フラグ

```bash
tov -interval 5
```

- `-interval <seconds>`: 自動更新間隔。`config.toml` より優先

---

## 画面構成

現行実装のレイアウトは 2 ペイン構成で、旧仕様にあったセッションサイドバーは存在しない。

```text
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

### 各エリア

| エリア | 内容 |
|--------|------|
| ヘッダー | タイトル、フィルター文字列、表示 pane 数、エラー |
| 上段 | agent pane 一覧 |
| 下段 | 選択 pane のプレビュー |
| フッター | キーバインド |

一覧の主要列:

- `DIRECTORY`: `gitutil.DisplayName()` によるディレクトリ名
- `STATUS`: 状態ラベル
- `DURATION`: 状態遷移からの経過時間

---

## ステータスモデル

実装上の主な状態は以下。

| 状態 | ラベル(en) | 意味 |
|------|------------|------|
| Registered | `📋 Registered` | セッション開始直後 |
| Running | `🤖 Running` | 処理中 |
| Waiting | `⏸ Waiting` | ユーザー確認待ち |
| Done | `✅ Done` | 完了 |
| Error | `❌ Error` | エラー |
| Unknown | `❓ Unknown` | 未判定 |

### 画面に表示する状態

TUI は以下だけを表示対象とする。

- Registered
- Running
- Waiting
- Done
- Error

`Idle` と `Unknown` は一覧に出さない。

### 状態遷移の基準

状態判定の主ソースは pane 出力ではなく **hook イベント** である。

#### 共通

| Event | 状態 |
|-------|------|
| `SessionStart` | Registered |
| `UserPromptSubmit` | Running |
| `PreToolUse` | Running |
| `Stop` | Done |

#### Claude Code

`Notification` イベントを追加で使う。

| NotificationType | 状態 |
|------------------|------|
| `permission_prompt` | Waiting |
| `elicitation_dialog` | Waiting |
| `idle_prompt` | Done |
| その他 | Waiting |

`SessionEnd` では状態ファイルを削除する。

#### Codex

Codex には `Notification` フックがないため、基本は共通遷移のみ。

ただしプレビュー末尾に以下のパターンが見えた場合は、画面表示上 `Waiting` に補正する。

- 権限確認プロンプト
- `Running ... hook`
- `... hook (completed)`

---

## プレビュー

- `tmux capture-pane -p -e -t <pane> -S -<N>` で末尾 N 行を取得する
- 行数 `N` は `display.preview_lines`
- 選択 pane の下段に最新行側を優先して表示する
- プレビューは表示用であり、通常の状態判定の主入力ではない

---

## フィルター

`/` でフィルターモードに入り、スペース区切りの AND 検索を行う。

検索対象:

- セッション名
- ウィンドウ名
- CWD
- ステータス文字列
- プレビュー本文

例:

```text
biwa running
```

これは `biwa` と `running` の両方を含む pane だけを残す。

---

## キーバインド

| キー | 動作 |
|------|------|
| `↑` / `k` | 上へ移動 |
| `↓` / `j` | 下へ移動 |
| `Enter` | 選択 pane にジャンプして終了 |
| `/` | フィルターモード開始 |
| `Esc` | フィルター解除、またはフィルターモード終了 |
| `r` | 即時リフレッシュ |
| `q` / `Ctrl+C` | 終了 |

補足:

- `Space` キー入力は受け取るが、現行 UI では見た目の変化はない
- `Tab` や `1`〜`9` による操作は未実装

---

## ジャンプ動作

TUI 終了後、選択された pane に対して tmux を切り替える。

### tmux 内から起動した場合

1. `select-window -t <session>:<window>`
2. `select-pane -t <pane-id>`
3. `switch-client -t <session>`

### tmux 外から起動した場合

- `attach-session -t <session>:<window>`

現行実装は tmux 外からのジャンプ時に `select-pane` を追加実行していない。

---

## 通知機能

macOS では hook イベントに応じて `terminal-notifier` で通知を送れる。

### 前提

- `terminal-notifier` が `PATH` にあること

### 通知対象

- Claude Code:
  - `Notification` -> `Claude Code - 確認`
  - `Stop` -> `Claude Code - 完了`
- Codex:
  - `Stop` -> `Codex - 完了`

### 通知クリック時の動作

通知から `tov focus` を呼び、最も最近アクティブだった tmux client を対象 pane に切り替える。

必要に応じてターミナルアプリも前面化する。

---

## 設定ファイル

パス: `~/.config/tov/config.toml`

```toml
[display]
interval = 2
preview_lines = 10
cwd_max_length = 40
language = "en"

[hook]
# state_dir = "/custom/path"

[notify]
enabled = true
# terminal_app = ""
# sound = ""
# icon = ""
```

### 各設定

| キー | 意味 |
|------|------|
| `display.interval` | 自動更新間隔（秒） |
| `display.preview_lines` | 取得するプレビュー行数 |
| `display.cwd_max_length` | 設定値は存在するが、現行表示では実質未使用 |
| `display.language` | `en` / `ja` |
| `hook.state_dir` | 状態ファイル保存先 |
| `notify.enabled` | 通知有効化 |
| `notify.terminal_app` | フォーカス時に前面化するターミナル名 |
| `notify.sound` | 通知音 |
| `notify.icon` | 通知アイコン |

---

## データ構造

### 状態ファイル

pane ごとに JSON を 1 つ持つ。

```go
type PaneState struct {
    PaneID          string
    Agent           string
    SessionName     string
    WindowIndex     int
    PaneIndex       int
    PID             int
    Status          Status
    StatusChangedAt time.Time
    LastEvent       string
    LastEventAt     time.Time
    Message         string
    TmuxSocket      string
}
```

### TUI 表示用 pane

```go
type Pane struct {
    ID           string
    Index        int
    CWD          string
    PID          int
    Width        int
    Height       int
    Active       bool
    Status       PaneStatus
    Duration     time.Duration
    Preview      []string
    Message      string
    Agent        string
    SessionName  string
    WindowIndex  int
    WindowName   string
    WindowActive bool
}
```

---

## ディレクトリ構成

```text
cmd/tov/main.go          CLI エントリポイント
internal/config/         config.toml 読み込み
internal/hook/           hook 処理、setup、通知、focus
internal/state/          状態ファイルの読み書き
internal/tmux/           tmux コマンド実行
internal/tui/            Bubble Tea UI
```

---

## 非機能要件

- `tmux` が `PATH` にない場合は明示的なエラーで終了する
- hook は tmux 外から誤って呼ばれても agent の処理を止めず静かに抜ける
- 状態ファイルは atomic rename で更新する
- stale な状態ファイルは TUI 更新時または `tov cleanup` で除去する

---

## 現行実装との差分メモ

以下は旧 spec にあったが現行実装には存在しない、または挙動が異なる項目。

- セッション一覧サイドバー
- 全 tmux pane を出力解析だけで分類する仕組み
- ステータス判定パターンの `config.toml` カスタマイズ
- `-s <session-name>` 起動オプション
- `Tab` / `1`〜`9` キーバインド
- tmux 外ジャンプ時の `attach-session ; select-window ; select-pane` の完全実行

