# tmux-overseer 仕様書

## 概要

tmuxの全セッション・ウィンドウ・ペインを一覧表示し、Claude Codeの実行状態を俯瞰しながら目的のペインへ素早くジャンプできるTUIツール。

コマンド名: `tov`（tmux overseer の略）

---

## 背景・解決したい問題

- 複数リポジトリを並行開発しており、tmuxセッションが常時10〜15個存在する
- 各ペインでClaude Codeが動いており、同時に5個程度が走り続けている
- 作業を離れたときに「どのペインが何をしているか」を俯瞰する手段がない
- 全ペインを手動で渡り歩いて状況確認するコストが高い
- 目的のペインを見つけてジャンプする操作を簡略化したい

---

## 技術スタック

| 項目 | 選定 |
|------|------|
| 言語 | Go |
| TUIライブラリ | [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) |
| tmux連携 | `tmux` コマンドのサブプロセス呼び出し（`display-message`, `list-panes`, `list-windows`, `list-sessions` など） |
| ビルド成果物 | シングルバイナリ `tov` |
| インストール | `go install` または `make install`（`~/.local/bin` へコピー） |

**Goを選定する理由:**
- シングルバイナリで配布できる
- bubbletea/lipglossによる本格的なTUI構築が容易
- tmuxのIPC（`tmux display-message -p`等）をサブプロセス経由で安全に扱える

---

## 起動・終了

```bash
# 基本起動（現在のtmuxセッション内から起動）
tov

# tmux外から起動（セッション一覧を表示）
tov

# 特定セッションを指定して起動
tov -s <session-name>
```

- tmux外から起動した場合もTUIは表示されるが、ジャンプ操作時は `tmux switch-client` または `tmux attach-session` を実行する
- tmux内から起動した場合は `tmux select-pane` / `tmux select-window` でジャンプする

---

## 画面構成

```
┌─────────────────────────────────────────────────────────────────┐
│  tov  │  Filter: [________]  │  Sessions: 3  Panes: 14         │
├──────────┬──────────────────────────────────────────────────────┤
│ SESSIONS │  PANE DETAIL                                         │
│          │                                                      │
│ ▶ work   │  Session : work                                      │
│   dev    │  Window  : 2:biwa-frontend                           │
│   misc   │  Pane    : %23  (active)                             │
│          │  CWD     : ~/src/biwa-frontend                       │
│          │  Size    : 220x50                                    │
│          │  PID     : 48291                                     │
│          │  Status  : 🤖 Claude running  (42s)                 │
│          │  Preview :                                           │
│          │  ┌────────────────────────────────┐                 │
│          │  │ > Analyzing component tree...  │                 │
│          │  │ ✓ Updated Button.tsx           │                 │
│          │  │ ✓ Updated index.ts             │                 │
│          │  │ ■ Writing tests...             │                 │
│          │  └────────────────────────────────┘                 │
├──────────┴──────────────────────────────────────────────────────┤
│  PANE LIST                                                      │
│                                                                 │
│  work:1:biwa-frontend     %21  🤖 Running   ~/src/biwa-front…  │
│  work:1:biwa-frontend     %22  ✅ Done      ~/src/biwa-front…  │
│▶ work:2:biwa-frontend     %23  🤖 Running   ~/src/biwa-front…  │
│  work:3:api-server        %24  💤 Idle      ~/src/api-server   │
│  dev:1:dashboard          %31  🤖 Running   ~/src/dashboard    │
│  dev:2:infra              %38  ❌ Error     ~/src/infra        │
│  misc:1:scratch           %45  💤 Idle      ~/scratch          │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  [↑↓] 移動   [Enter] ジャンプ   [/] フィルター   [r] 更新      │
│  [Space] プレビュー展開   [q] 終了                              │
└─────────────────────────────────────────────────────────────────┘
```

### レイアウト説明

| エリア | 役割 |
|--------|------|
| ヘッダー | フィルター入力欄、統計情報（セッション数・ペイン数） |
| 左ペイン（Sessions） | セッション一覧。選択でペインリストを絞り込む |
| 右上（Pane Detail） | 選択中ペインの詳細情報とプレビュー |
| 下（Pane List） | 全ペインの一覧。ここで移動・選択を行う |
| フッター | キーバインド一覧 |

---

## ペインステータス検出

tmuxペインの最終出力内容を `tmux capture-pane -p` で取得し、以下のパターンで状態を判定する。

| ステータス | 表示 | 判定ルール |
|-----------|------|-----------|
| Claude実行中 | 🤖 Running | 末尾行が Claude Code の実行中パターン（後述）にマッチ |
| Claude完了 | ✅ Done | Claude Code が終了プロンプトを表示している状態 |
| エラー | ❌ Error | Claude Code がエラーメッセージを出力して停止している |
| アイドル | 💤 Idle | シェルプロンプト（`$`, `%`, `❯` 等）が末尾にある |
| 不明 | ❓ Unknown | 上記いずれにもマッチしない |

### Claude Code 実行中パターン（正規表現）

```
# 実行中を示す行パターン（いずれかにマッチ = Running）
- `^[◐◑◒◓⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]` （スピナー文字）
- `Analyzing`
- `Writing`
- `Reading`
- `Executing`
- `Running`
- `Thinking`

# 完了を示す行パターン
- `> ` で始まる入力待ちプロンプト（Claude Code のプロンプト）
- `✓ Task completed`

# エラーを示す行パターン
- `Error:`
- `Failed to`
- `✗`
```

※ パターンは設定ファイルでカスタマイズ可能にする（後述）

---

## キーバインド

| キー | 動作 |
|------|------|
| `↑` / `↓` | ペインリストのカーソル移動 |
| `k` / `j` | 同上（Vim風） |
| `Enter` | 選択ペインへジャンプ（tmux switch） |
| `/` | フィルターモードに入る |
| `Esc` | フィルタークリア / モード終了 |
| `r` | 手動リフレッシュ |
| `Space` | プレビューエリアの展開/折りたたみ |
| `Tab` | セッション一覧とペインリストのフォーカス切替 |
| `1`〜`9` | セッション番号で直接絞り込み |
| `q` / `Ctrl+C` | 終了 |

---

## フィルター機能

`/` キーで入力モードに入り、以下の対象に対してインクリメンタルサーチを行う。

- セッション名
- ウィンドウ名
- ペインのCWD（カレントディレクトリ）
- ペインの最終出力テキスト

フィルターはOR条件ではなくAND条件（スペース区切りで複数ワード指定可能）。

```
Filter: biwa running
→ CWD が biwa を含み、かつステータスが Running のペインのみ表示
```

---

## 自動更新

- デフォルト: 2秒ごとに自動リフレッシュ
- `--interval <秒>` オプションで変更可能
- `r` キーで即時リフレッシュ

---

## ジャンプ動作

`Enter` でペインを選択したとき:

1. **tmux内から起動した場合**
   ```
   tmux select-window -t <session>:<window>
   tmux select-pane -t <pane-id>
   ```
   → tovを終了し、対象ペインがアクティブになる

2. **tmux外から起動した場合**
   ```
   tmux attach-session -t <session> \; select-window -t <window> \; select-pane -t <pane-id>
   ```

---

## 設定ファイル

`~/.config/tov/config.toml`

```toml
[display]
# 自動更新間隔（秒）
interval = 2

# プレビューの最大行数
preview_lines = 10

# ペインリストの1行に表示するCWDの最大文字数
cwd_max_length = 40

[status]
# ステータス判定のカスタムパターン（正規表現）
running_patterns = [
  "^[◐◑◒◓⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]",
  "Analyzing",
  "Writing",
  "Reading",
  "Executing",
]

done_patterns = [
  "^> $",
  "✓ Task completed",
]

error_patterns = [
  "^Error:",
  "Failed to",
  "^✗",
]

[keybinds]
# 将来的なカスタムキーバインド設定用（v1では固定）
```

---

## ディレクトリ構成

```
tov/
├── main.go
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── internal/
│   ├── tmux/
│   │   ├── client.go       # tmuxコマンド呼び出し
│   │   ├── session.go      # Session/Window/Pane 構造体
│   │   └── status.go       # ペインステータス判定
│   ├── config/
│   │   └── config.go       # 設定ファイル読み込み
│   └── tui/
│       ├── model.go        # bubbletea Model
│       ├── update.go       # bubbletea Update
│       ├── view.go         # bubbletea View (lipgloss)
│       ├── filter.go       # フィルターロジック
│       └── keys.go         # キーバインド定義
└── cmd/
    └── tov/
        └── main.go
```

---

## データ構造

```go
type Session struct {
    Name    string
    Windows []Window
}

type Window struct {
    Index  int
    Name   string
    Panes  []Pane
    Active bool
}

type Pane struct {
    ID       string   // %23 形式
    Index    int
    CWD      string
    PID      int
    Width    int
    Height   int
    Active   bool
    Status   PaneStatus
    Duration time.Duration // Runningの場合の経過時間
    Preview  []string      // capture-pane の末尾N行
}

type PaneStatus int

const (
    StatusUnknown PaneStatus = iota
    StatusRunning
    StatusDone
    StatusError
    StatusIdle
)
```

---

## 実装フェーズ

### Phase 1（MVP）
- [ ] tmuxセッション・ウィンドウ・ペイン情報の取得
- [ ] ペインステータス判定（Running / Idle / Done / Error）
- [ ] ペインリスト表示（bubbletea + lipgloss）
- [ ] カーソル移動とEnterキーでのジャンプ
- [ ] 自動リフレッシュ（2秒）

### Phase 2
- [ ] セッションサイドバー
- [ ] ペイン詳細パネル（プレビュー表示）
- [ ] インクリメンタルフィルター（`/`キー）
- [ ] 設定ファイル対応

### Phase 3
- [ ] Running状態の経過時間表示
- [ ] エラーペインのハイライト
- [ ] `go install` によるインストール対応
- [ ] README整備

---

## 非機能要件

- tmuxが存在しない環境での起動時は明示的なエラーメッセージを表示して終了する
- ペイン数が50を超えても描画がもたつかないこと（仮想スクロール不要、ただし100ms以内に描画更新されること）
- `capture-pane` 失敗時はステータスを `Unknown` として続行する（クラッシュしない）
- macOS / Linux で動作すること

---

## インストール方法（完成後）

```bash
# リポジトリをクローン
git clone https://github.com/<user>/tov.git
cd tov

# ビルド & インストール
make install
# または
go install ./cmd/tov@latest
```

---

## Claude Codeへの補足指示

- `bubbletea` の `tea.Tick` を使って2秒ごとの自動更新を実装すること
- tmuxコマンドの呼び出しはすべて `internal/tmux/client.go` に集約し、他パッケージからはインターフェース経由で利用すること
- ステータス判定のパターンは `internal/tmux/status.go` に定数として定義し、設定ファイルでオーバーライドできるようにすること
- Phase 1 から動くMVPを先に完成させ、その後 Phase 2 を追加すること
- `lipgloss` のスタイル定義は `internal/tui/view.go` の先頭にまとめて定義すること
- エラーハンドリングは `fmt.Errorf` でラップして上位に伝播させること（`log.Fatal` はmain以外では使わない）
