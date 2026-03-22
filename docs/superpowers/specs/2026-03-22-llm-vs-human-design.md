# LLM vs Human: Claude Code 対戦機能

## Overview

Claude Code から MCP サーバー経由で Yahtzee の対戦に参加できるようにする。
人間は TUI、Claude は MCP ツールでプレイし、同じ Game Server に接続して対戦する。
Claude は対戦中にチャットで実況・解説を送信でき、人間の TUI にも表示される。

## Motivation

- 現在の MCP サーバーはローカル AI 対戦のみ。人間との対戦ができない
- 既存の P2P は NAT 越えできず、インターネット対戦が実質不可能
- Claude Code を対戦相手にすることで、localhost で完結する面白い対戦体験を実現
- 将来的に Claude Code 同士の対戦にも拡張可能

## Architecture

```
Phase 1 (Human vs Claude Code):

[Human TUI]  ──TCP──→  [Game Server (yatz serve)]  ←──TCP──  [MCP Server ← Claude Code]
  (yatz join)              localhost:9876                        (yatz mcp)

Phase 2 (Claude Code vs Claude Code):

[Claude Code A → MCP A]  ──TCP──→  [Game Server]  ←──TCP──  [Claude Code B → MCP B]
```

### Game Server (`p2p/server.go`)

ヘッドレスなゲームサーバー。ゲームエンジンを保持し、全クライアントを対等に扱う。
既存の `p2p/` パッケージ内に配置し、ネットワーキングコードを一箇所に集約する。

**責務:**
- TCP リッスン、指定人数のクライアント接続受付
- ハンドシェイク処理、プレイヤー ID 割り当て（接続順に `player-0`, `player-1`）
- 全クライアント接続後 `game_start` をブロードキャスト
- ターン管理: 現在プレイヤーのクライアントから `action` を読み取り、全クライアントに `state_update` をブロードキャスト
- `chat` メッセージの受信・ブロードキャスト
- ゲーム終了時 `game_over` をブロードキャスト
- クライアント切断時のクリーンアップとエラー通知

**コマンド:**
```bash
yatz serve --port 9876 --players 2
```

**ターン管理の詳細:**

サーバーはシングルゴルーチンでゲームループを回す。現在のプレイヤーの接続から `action` を読み取り、エンジンに適用、全クライアントに `state_update` をブロードキャストする。`score` アクション後、次のプレイヤーに `turn_start` を送信する。`chat` メッセージは各クライアント接続に対する読み取りゴルーチンで受信し、即座に全クライアントにブロードキャストする（ゲームループとは独立）。

```
読み取りゴルーチン (per client):
  loop:
    read message from conn
    if MsgChat: broadcast to all clients
    if MsgAction: forward to actionCh (per-client channel)
  end loop

メインゴルーチン (ゲームループ):
  for each round:
    for each player:
      send turn_start to current player
      loop:
        read action from current player's actionCh  ← connではなくチャネルから読む
        apply to engine
        broadcast state_update to all
        if action == score: break (next player)
      end loop
    end for
  end for
  broadcast game_over
```

注: `turn_start` は既存の `MsgTurnStart` メッセージ型を再利用する。

**接続フロー:**
1. Client A connects → handshake("Alice") → server responds with handshake + assigned player ID
2. Client B connects → handshake("Claude") → server responds with handshake + assigned player ID
3. Server → broadcast `game_start` to both (payload includes full game state)
4. Server → send `turn_start` to player-0
5. Wait for `action` from player-0 → apply to engine → broadcast `state_update`
6. On `score` action → send `turn_start` to player-1
7. On game finish → broadcast `game_over`

**クライアント切断時の動作:**
- サーバーは切断を検知し、残りのクライアントに `error` メッセージを送信してから接続を閉じる
- ゲームは中断される（再接続は Phase 1 では非スコープ）

### Handshake Protocol Extension

現在のハンドシェイクはクライアント→サーバーの名前交換のみ。サーバーからの応答にプレイヤー ID を含めるよう拡張する。

**現在:**
```
Client → Server: { type: "handshake", payload: { name: "Alice" } }
Server → Client: { type: "handshake", payload: { name: "Server" } }
```

**拡張後:**
```
Client → Server: { type: "handshake", payload: { name: "Alice" } }
Server → Client: { type: "handshake", payload: { name: "Server", player_id: "player-0" } }
```

`HandshakePayload` に `PlayerID string` フィールドを追加。既存の P2P (`yatz host/join`) では `PlayerID` は空文字になるため後方互換性あり（ゲスト側は従来通り `player-1` 固定でフォールバック）。

```go
type HandshakePayload struct {
    Name     string `json:"name"`
    PlayerID string `json:"player_id,omitempty"`
}
```

`RemoteClient` 側の変更: `newRemoteClientFromConn()` でサーバーからのハンドシェイク応答に `PlayerID` が含まれていれば、それを使用。空であれば従来通り `player-1` にフォールバック。

### MCP Server Changes (`mcp/server.go`)

**新しいツール:**

| Tool | Parameters | Description |
|------|-----------|-------------|
| `join_game` | `addr: string`, `name: string` | Game Server に TCP 接続 |
| `send_chat` | `text: string` | チャットメッセージ送信（実況・解説用）。`join_game` 後のみ使用可能 |

**既存ツールの変更:**
- `gameServer.client` の型を `*engine.LocalClient` → `engine.GameClient` に変更
- `gameServer.conn` フィールドを追加（`join_game` 時の TCP 接続を保持、チャット送信用）
- `new_game` と `join_game` は排他的モード。`join_game` 後に `new_game` を呼ぶと接続を切断してローカルモードに戻る
- `send_chat` は `join_game` 後のみ有効。それ以外では "Not connected to a game server" エラーを返す

**`handleScore` の修正:**
- 現在 `gs.game.Dice` を直接参照してスコアを計算しているが、`RemoteClient` 使用時は `gs.game` が nil
- `gs.client.GetState()` から dice を取得するよう変更
- **重要:** `GetState()` は `Score()` の **前に** 呼ぶこと。`Score()` はターンを進めてダイスをリセットするため、`Score()` 後の `GetState()` では正しいダイス値が取れない

**`new_game` / `join_game` の排他性:**
- `join_game` 後は `gs.game` を `nil` にセットし、ローカルゲームへの誤参照を防ぐ
- `new_game` 呼び出し時に既存の TCP 接続があれば切断してからローカルモードに戻る

**`RemoteClient.Score()` のブロッキング動作:**
- `Score()` は相手のターンが終わるまでブロックし、次の自分のターン開始時に返る
- MCP のツール呼び出しは同期的なので、この動作と自然に噛み合う
- Claude Code は `score` ツールの結果を受け取った時点で次の自分のターンが来ている

### Protocol Extension (`p2p/protocol.go`)

**追加メッセージ型:**

```go
MsgChat = "chat"

type ChatPayload struct {
    PlayerID string `json:"player_id"`
    Name     string `json:"name"`
    Text     string `json:"text"`
}
```

**追加関数:**
- `NewChatMsg(playerID, name, text string) *Message`
- `DecodeChat(msg *Message) (*ChatPayload, error)`

**`HandshakePayload` の拡張:**
- `PlayerID string` フィールドを追加（`omitempty` で後方互換）

既存メッセージ型のフォーマット変更なし。

### RemoteClient Changes (`p2p/guest.go`)

- `playerID` のハードコード (`"player-1"`) を削除。ハンドシェイク応答の `PlayerID` を使用（空なら `"player-1"` にフォールバック）
- `chatCh chan *ChatPayload` チャネルを追加（バッファサイズ 16、満杯時はドロップ）
- `listen()` の `switch` 文に `MsgChat` ケースを追加し、`chatCh` に送信
- TUI が `chatCh` を購読してチャットメッセージを表示

### TUI Changes (`cli/model.go`)

- 画面下部にチャット表示エリア（直近 3〜5 件）
- `RemoteClient` の `chatCh` から `tea.Msg` としてチャットメッセージを受信
- 人間側からのチャット送信は Phase 1 では非スコープ

**表示イメージ:**
```
Round: 3/13 | You | Roll: 1/3
Dice: [3] [3] [5] [2] [6]

  ones        -     three_of_a_kind  -
  twos        -     four_of_a_kind   -
  ...

─── Chat ───────────────────────────
Claude: いい出目だね、スリーカード狙えるかも
Claude: 5と6捨ててリロールしよう
```

## Typical Usage

### Human vs Claude Code

```bash
# Terminal 1: Start game server
yatz serve --port 9876 --players 2

# Terminal 2: Human joins as TUI player
yatz join localhost:9876 --name Alice

# Claude Code: Claude joins via MCP
> join_game(addr: "localhost:9876", name: "Claude")
> roll_dice()
> send_chat(text: "お、いい出目！フルハウス狙うか")
> hold_dice(indices: [0, 1])
> score(category: "full_house")
```

### Claude Code vs Claude Code (Phase 2)

```bash
# Terminal 1: Start game server
yatz serve --port 9876 --players 2

# Claude Code A:
> join_game(addr: "localhost:9876", name: "Claude-A")

# Claude Code B:
> join_game(addr: "localhost:9876", name: "Claude-B")
```

Phase 2 では Game Server 側の変更は不要。

## New Files

| File | Description |
|------|-------------|
| `p2p/server.go` | ヘッドレス Game Server |
| `cmd/yatz/serve.go` | `yatz serve` サブコマンド |

## Modified Files

| File | Change |
|------|--------|
| `p2p/protocol.go` | `MsgChat`, `ChatPayload`, `NewChatMsg`, `DecodeChat` 追加。`HandshakePayload` に `PlayerID` 追加 |
| `p2p/guest.go` | `RemoteClient` の `playerID` をハンドシェイクから取得。`chatCh` 追加、`listen()` でチャットハンドリング |
| `mcp/server.go` | `client` を `GameClient` に変更。`handleScore` の dice 取得修正。`join_game`/`send_chat` ツール追加 |
| `cli/model.go` | チャット表示エリア追加、チャット受信ハンドリング |
| `cmd/yatz/main.go` | `serve` サブコマンド登録 |

## Testing Strategy

- **`p2p/server_test.go`**: ヘッドレスサーバーの単体テスト（接続、ハンドシェイク、ターン管理、チャットブロードキャスト）。既存の `p2p/host_test.go` のパターンを踏襲
- **`p2p/server_e2e_test.go`**: 2クライアント接続でフルゲームを回す E2E テスト。`-short` フラグでスキップ
- **`mcp/server_test.go`**: `join_game` → `roll_dice` → `score` のフローテスト（テスト用にローカルサーバーを起動）
- **チャットテスト**: サーバー経由でチャットメッセージが全クライアントに届くことを検証
- **後方互換テスト**: `HandshakePayload` 変更後も既存の `yatz host` + `yatz join` の組み合わせが動作することを検証（`PlayerID` 空文字時のフォールバック）

## Scope

### In Scope (Phase 1)
- ヘッドレス Game Server (`yatz serve`)
- ハンドシェイクプロトコル拡張（プレイヤー ID 割り当て）
- MCP に `join_game`, `send_chat` ツール追加
- MCP `handleScore` の dice 取得修正
- プロトコルに `MsgChat` 追加
- `RemoteClient` にチャットチャネル追加、`playerID` 動的割り当て
- TUI にチャット表示エリア追加
- 既存 `yatz join` でサーバー接続可能（ハンドシェイク後方互換）

### Out of Scope
- 人間側からのチャット送信
- 3 人以上の対戦
- 観戦モード
- AI 戦略の改善
- 再接続 / 状態リカバリ
- 既存 `yatz play`（ローカル AI 対戦）の変更
- 既存 `yatz host`（P2P ホスト）の変更
