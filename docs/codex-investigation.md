# Codex CLI 調査結果（tov統合に向けて）

調査日: 2026-03-31

## 概要

OpenAI Codex CLIはClaude Codeとほぼ同じライフサイクルフック機構を持っており、tovでのサポートは実現可能。

- **リポジトリ**: github.com/openai/codex
- **コマンド名**: `codex`
- **インストール**: `npm i -g @openai/codex` または `brew install --cask codex`
- **実装言語**: Rust（codex-rs/）+ Node.js ラッパー
- **TUIフレームワーク**: ratatui

## フックイベント比較

| Codex | Claude Code | tov ステータスマッピング |
|---|---|---|
| `SessionStart` | `SessionStart` | `registered` |
| `UserPromptSubmit` | `UserPromptSubmit` | `running` |
| `PreToolUse` | `PreToolUse` | `running` |
| `PostToolUse` | `PostToolUse` | （未使用） |
| `Stop` | `Stop` | `done` |
| **なし** | `Notification` | `waiting` / `done` |
| **なし** | `SessionEnd` | state file削除 |

## フック設定

### 設定ファイルの場所

- Claude Code: `~/.claude/settings.json` 内の `hooks` オブジェクト
- Codex: `~/.codex/hooks.json`（専用ファイル）

### Codex hooks.json フォーマット

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": null,
        "hooks": [
          {
            "type": "command",
            "command": "tov hook SessionStart",
            "timeout": 600,
            "async": false,
            "statusMessage": "running hook..."
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": null,
        "hooks": [
          { "type": "command", "command": "tov hook UserPromptSubmit" }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": null,
        "hooks": [
          { "type": "command", "command": "tov hook PreToolUse" }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": null,
        "hooks": [
          { "type": "command", "command": "tov hook Stop" }
        ]
      }
    ]
  }
}
```

### Claude Code settings.json との構造差異

- Claude Code: `matcher` は空文字列 `""`
- Codex: `matcher` は `null`
- Codex固有フィールド: `async`, `statusMessage`
- フックハンドラタイプ: `command`（共通）、`prompt`/`agent`（Codex固有、未実装）

## stdin JSON スキーマ

### 共通フィールド

```json
{
  "session_id": "string",
  "cwd": "/path/to/dir",
  "hook_event_name": "SessionStart|UserPromptSubmit|PreToolUse|PostToolUse|Stop",
  "model": "string",
  "permission_mode": "default|acceptEdits|plan|dontAsk|bypassPermissions"
}
```

### Codex固有フィールド

- `turn_id`: ターンスコープのイベント（PreToolUse, PostToolUse, Stop）に付与
- `transcript_path`: セッショントランスクリプトファイルのパス（nullable）
- `source`: SessionStartのみ。`"startup"` または `"resume"`

### Claude Code固有フィールド

- `notification_type`: Notificationイベントのみ。`"permission_prompt"` / `"idle_prompt"` / `"elicitation_dialog"`

## フック出力スキーマ

### Codex

stdout にJSON出力:
```json
{
  "continue": true,
  "stopReason": null,
  "suppressOutput": false,
  "systemMessage": null,
  "decision": "block",
  "reason": "..."
}
```

終了コード:
- 0: 成功（stdoutのJSONをパース）
- 2: ブロック/フィードバック（stderrを理由として使用）
- その他: 失敗

### Claude Code

同様のJSON出力（`decision` フィールドなど共通）

## 環境変数

| 変数 | Codex | Claude Code | 用途 |
|---|---|---|---|
| `$TMUX_PANE` | ○ | ○ | tmuxペインID（`%23`等） |
| `$TMUX` | ○ | ○ | tmuxソケットパス |
| `CODEX_THREAD_ID` | ○ | - | Codexセッション/スレッドID |
| `CODEX_HOME` | ○ | - | Codexデータディレクトリ（デフォルト `~/.codex`） |

## 状態管理

- Claude Code: なし（tovが自前でJSONファイル管理）
- Codex: SQLite（`$CODEX_HOME/state_v5.db`）— ただしtovでは使用しない（フック経由で十分）

## tmux検出

Codexは `codex-rs/terminal-detection/` で明示的にtmuxを検出している（`tmux display-message` で `client_termtype` を取得）。tmux内での動作は想定されている。

## 実装上の課題と対策

### 1. Notificationイベントがない

**課題**: Codexには `Notification` フックがないため、`waiting`（権限確認待ち）ステータスを直接検出できない。

**対策案**:
- `PreToolUse` フックの戻り値でブロック判定を行う方法は、tov側では制御しないため不適
- Codexでは `waiting` ステータスを使わず、`running` → `done` の2状態遷移で運用
- または `capture-pane` で権限確認UIのパターンマッチ（フォールバック）

### 2. SessionEndイベントがない

**課題**: Codexにはセッション終了を通知するフックがない。state fileが残り続ける可能性。

**対策**: 既存のstaleクリーンアップ機構（tmuxペイン存在チェック + `RemoveStale()`）で対応済み。追加対応不要。

### 3. 設定ファイル形式の違い

**課題**: `tov setup` がClaude Code向けの `settings.json` 書き込みのみ対応。

**対策**: `tov setup --codex` フラグを追加し、`~/.codex/hooks.json` への書き込みロジックを実装。

## 推奨実装方針

### handler.go の変更

`HandleEvent` は入力JSONの共通フィールド（`session_id`, `cwd`, `hook_event_name`）で動作するため、Codex/Claude Code共通で使える。主な分岐:

- `Notification` イベント: Claude Code専用（Codexでは発火しない）
- `SessionEnd` イベント: Claude Code専用（Codexでは発火しない）
- `Stop` イベント: 共通（両方とも `done` にマッピング）

### setup.go の変更

- `tov setup` — Claude Code向け（既存）
- `tov setup --codex` — Codex向け（`~/.codex/hooks.json` に書き込み）
- `tov setup --all` — 両方に設定

### ステータスマッピング（Codex）

| イベント | ステータス遷移 |
|---|---|
| SessionStart | → `registered` |
| UserPromptSubmit | → `running` |
| PreToolUse | → `running`（変化なし） |
| Stop | → `done` |

`waiting` は使わない（Codexでは検出不可）。

## 参考リンク

- Codex CLI リポジトリ: https://github.com/openai/codex
- フック実装: `codex-rs/hooks/src/`
- フックイベント定義: `codex-rs/hooks/src/events/`
- 設定ドキュメント: `docs/config.md`（リポジトリ内）
