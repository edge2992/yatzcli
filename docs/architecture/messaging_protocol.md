# メッセージプロトコル設計書

## 概要

本ドキュメントはYatzCLIのクライアント・サーバー間通信プロトコルを定義します。JSONフォーマットを採用し、Command/Eventパターンに基づいた明確なメッセージ構造を提供します。

## プロトコル原則

### 1. JSONベースの通信

- **可読性**: デバッグが容易
- **拡張性**: 新しいフィールドの追加が容易
- **互換性**: 多言語対応が可能（将来的なWebクライアント対応）
- **標準化**: 広く使用される標準フォーマット

### 2. Command/Event分離

```
┌─────────┐                  ┌─────────┐
│ Client  │  ── Command ──→  │ Server  │
│         │                  │         │
│         │  ←── Event ────  │         │
│         │  ←── Response ─  │         │
└─────────┘                  └─────────┘
```

- **Command**: クライアントからの要求（意図）
- **Event**: サーバーからの通知（事実）
- **Response**: コマンドの実行結果

### 3. メッセージ識別

- `messageType`: メッセージの種類を識別
- `messageId`: 各メッセージの一意識別子（UUID）
- `timestamp`: メッセージの生成時刻（ISO 8601形式）
- `correlationId`: リクエスト/レスポンスの関連付け

---

## 基本メッセージ構造

### ベースメッセージフォーマット

```json
{
  "messageType": "string",
  "messageId": "uuid",
  "timestamp": "ISO8601",
  "correlationId": "uuid",
  "payload": {}
}
```

### TypeScript型定義（参考）

```typescript
interface BaseMessage {
  messageType: string;
  messageId: string;
  timestamp: string;
  correlationId?: string;
  payload: unknown;
}

interface Command extends BaseMessage {
  messageType: `${string}Command`;
}

interface Event extends BaseMessage {
  messageType: `${string}Event`;
}

interface Response extends BaseMessage {
  messageType: 'Success' | 'Error';
  success: boolean;
}
```

### Go型定義

```go
package protocol

type Message struct {
    MessageType   string          `json:"messageType"`
    MessageID     string          `json:"messageId"`
    Timestamp     string          `json:"timestamp"`
    CorrelationID string          `json:"correlationId,omitempty"`
    Payload       json.RawMessage `json:"payload"`
}

type MessageType string

const (
    // Commands
    MessageTypeCreateRoom    MessageType = "CreateRoomCommand"
    MessageTypeJoinRoom      MessageType = "JoinRoomCommand"
    MessageTypeStartGame     MessageType = "StartGameCommand"
    MessageTypeRollDice      MessageType = "RollDiceCommand"
    MessageTypeChooseCategory MessageType = "ChooseCategoryCommand"

    // Events
    MessageTypeRoomCreated   MessageType = "RoomCreatedEvent"
    MessageTypePlayerJoined  MessageType = "PlayerJoinedEvent"
    MessageTypeGameStarted   MessageType = "GameStartedEvent"
    MessageTypeTurnStarted   MessageType = "TurnStartedEvent"
    MessageTypeDiceRolled    MessageType = "DiceRolledEvent"

    // Responses
    MessageTypeSuccess       MessageType = "Success"
    MessageTypeError         MessageType = "Error"
)
```

---

## コマンド（Command）

クライアントからサーバーへの要求メッセージ。

### 1. CreateRoomCommand

**目的**: 新しいゲームルームを作成

```json
{
  "messageType": "CreateRoomCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-13T10:30:00Z",
  "payload": {
    "playerName": "Alice",
    "maxPlayers": 4,
    "roomSettings": {
      "allowReconnect": true,
      "turnTimeout": 60
    }
  }
}
```

**Payload定義**:
```go
type CreateRoomPayload struct {
    PlayerName   string        `json:"playerName" validate:"required,min=1,max=20"`
    MaxPlayers   int           `json:"maxPlayers" validate:"min=2,max=4"`
    RoomSettings *RoomSettings `json:"roomSettings,omitempty"`
}

type RoomSettings struct {
    AllowReconnect bool `json:"allowReconnect"`
    TurnTimeout    int  `json:"turnTimeout"` // 秒
}
```

### 2. JoinRoomCommand

**目的**: 既存のルームに参加

```json
{
  "messageType": "JoinRoomCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2025-11-13T10:31:00Z",
  "payload": {
    "roomId": "room-123",
    "playerName": "Bob"
  }
}
```

**Payload定義**:
```go
type JoinRoomPayload struct {
    RoomID     string `json:"roomId" validate:"required"`
    PlayerName string `json:"playerName" validate:"required,min=1,max=20"`
}
```

### 3. LeaveRoomCommand

**目的**: ルームから退出

```json
{
  "messageType": "LeaveRoomCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2025-11-13T10:32:00Z",
  "payload": {
    "roomId": "room-123"
  }
}
```

### 4. StartGameCommand

**目的**: ゲーム開始（ホストのみ）

```json
{
  "messageType": "StartGameCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440003",
  "timestamp": "2025-11-13T10:33:00Z",
  "payload": {
    "roomId": "room-123"
  }
}
```

### 5. RollDiceCommand

**目的**: ダイスをロール

```json
{
  "messageType": "RollDiceCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440004",
  "timestamp": "2025-11-13T10:34:00Z",
  "payload": {
    "roomId": "room-123",
    "heldDiceIndices": [0, 2, 4]
  }
}
```

**Payload定義**:
```go
type RollDicePayload struct {
    RoomID          string `json:"roomId" validate:"required"`
    HeldDiceIndices []int  `json:"heldDiceIndices" validate:"dive,min=0,max=4"`
}
```

### 6. ChooseCategoryCommand

**目的**: スコアカテゴリーを選択

```json
{
  "messageType": "ChooseCategoryCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440005",
  "timestamp": "2025-11-13T10:35:00Z",
  "payload": {
    "roomId": "room-123",
    "category": "ThreeOfAKind"
  }
}
```

**Payload定義**:
```go
type ChooseCategoryPayload struct {
    RoomID   string `json:"roomId" validate:"required"`
    Category string `json:"category" validate:"required,oneof=Ones Twos Threes Fours Fives Sixes ThreeOfAKind FourOfAKind FullHouse SmallStraight LargeStraight Yahtzee Chance"`
}
```

### 7. ListRoomsCommand

**目的**: 利用可能なルーム一覧を取得

```json
{
  "messageType": "ListRoomsCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440006",
  "timestamp": "2025-11-13T10:36:00Z",
  "payload": {
    "filter": {
      "state": "WaitingForPlayers",
      "hasSpace": true
    }
  }
}
```

---

## イベント（Event）

サーバーからクライアントへの通知メッセージ。

### 1. RoomCreatedEvent

**目的**: ルームが作成されたことを通知

```json
{
  "messageType": "RoomCreatedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-13T10:30:01Z",
  "correlationId": "550e8400-e29b-41d4-a716-446655440000",
  "payload": {
    "roomId": "room-123",
    "hostPlayerId": "player-001",
    "hostPlayerName": "Alice",
    "maxPlayers": 4
  }
}
```

**Payload定義**:
```go
type RoomCreatedPayload struct {
    RoomID         string `json:"roomId"`
    HostPlayerID   string `json:"hostPlayerId"`
    HostPlayerName string `json:"hostPlayerName"`
    MaxPlayers     int    `json:"maxPlayers"`
}
```

### 2. PlayerJoinedEvent

**目的**: プレイヤーがルームに参加したことを全員に通知

```json
{
  "messageType": "PlayerJoinedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2025-11-13T10:31:01Z",
  "payload": {
    "roomId": "room-123",
    "playerId": "player-002",
    "playerName": "Bob",
    "currentPlayerCount": 2
  }
}
```

### 3. PlayerLeftEvent

**目的**: プレイヤーがルームから退出したことを通知

```json
{
  "messageType": "PlayerLeftEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2025-11-13T10:32:01Z",
  "payload": {
    "roomId": "room-123",
    "playerId": "player-002",
    "playerName": "Bob",
    "reason": "voluntary",
    "currentPlayerCount": 1
  }
}
```

### 4. GameStartedEvent

**目的**: ゲームが開始されたことを全員に通知

```json
{
  "messageType": "GameStartedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440003",
  "timestamp": "2025-11-13T10:33:01Z",
  "payload": {
    "roomId": "room-123",
    "players": [
      {
        "playerId": "player-001",
        "playerName": "Alice",
        "order": 0
      },
      {
        "playerId": "player-002",
        "playerName": "Bob",
        "order": 1
      }
    ],
    "firstPlayerId": "player-001"
  }
}
```

### 5. TurnStartedEvent

**目的**: 新しいターンが開始されたことを通知

```json
{
  "messageType": "TurnStartedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440004",
  "timestamp": "2025-11-13T10:34:01Z",
  "payload": {
    "roomId": "room-123",
    "playerId": "player-001",
    "playerName": "Alice",
    "turnNumber": 1,
    "roundNumber": 1
  }
}
```

### 6. DiceRolledEvent

**目的**: ダイスがロールされたことを全員に通知

```json
{
  "messageType": "DiceRolledEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440005",
  "timestamp": "2025-11-13T10:34:02Z",
  "payload": {
    "roomId": "room-123",
    "playerId": "player-001",
    "diceSet": [
      {"value": 3, "held": false},
      {"value": 3, "held": false},
      {"value": 3, "held": false},
      {"value": 2, "held": false},
      {"value": 5, "held": false}
    ],
    "rollCount": 1,
    "canRollAgain": true
  }
}
```

**Payload定義**:
```go
type DiceRolledPayload struct {
    RoomID       string `json:"roomId"`
    PlayerID     string `json:"playerId"`
    DiceSet      []Dice `json:"diceSet"`
    RollCount    int    `json:"rollCount"`
    CanRollAgain bool   `json:"canRollAgain"`
}

type Dice struct {
    Value int  `json:"value"`
    Held  bool `json:"held"`
}
```

### 7. CategoryChosenEvent

**目的**: カテゴリーが選択され、スコアが記録されたことを通知

```json
{
  "messageType": "CategoryChosenEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440006",
  "timestamp": "2025-11-13T10:35:01Z",
  "payload": {
    "roomId": "room-123",
    "playerId": "player-001",
    "category": "ThreeOfAKind",
    "score": 11,
    "diceSet": [
      {"value": 3, "held": false},
      {"value": 3, "held": false},
      {"value": 3, "held": false},
      {"value": 1, "held": false},
      {"value": 1, "held": false}
    ],
    "totalScore": 11
  }
}
```

### 8. ScoreboardUpdatedEvent

**目的**: スコアボードが更新されたことを通知

```json
{
  "messageType": "ScoreboardUpdatedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440007",
  "timestamp": "2025-11-13T10:35:02Z",
  "payload": {
    "roomId": "room-123",
    "scoreboard": [
      {
        "playerId": "player-001",
        "playerName": "Alice",
        "scores": {
          "Ones": null,
          "Twos": null,
          "ThreeOfAKind": 11
        },
        "upperSectionTotal": 0,
        "upperBonus": 0,
        "lowerSectionTotal": 11,
        "totalScore": 11
      },
      {
        "playerId": "player-002",
        "playerName": "Bob",
        "scores": {},
        "totalScore": 0
      }
    ]
  }
}
```

### 9. GameFinishedEvent

**目的**: ゲームが終了したことを通知

```json
{
  "messageType": "GameFinishedEvent",
  "messageId": "650e8400-e29b-41d4-a716-446655440008",
  "timestamp": "2025-11-13T11:00:00Z",
  "payload": {
    "roomId": "room-123",
    "rankings": [
      {
        "rank": 1,
        "playerId": "player-001",
        "playerName": "Alice",
        "totalScore": 245
      },
      {
        "rank": 2,
        "playerId": "player-002",
        "playerName": "Bob",
        "totalScore": 198
      }
    ],
    "winner": {
      "playerId": "player-001",
      "playerName": "Alice",
      "totalScore": 245
    }
  }
}
```

---

## レスポンス（Response）

コマンドの実行結果を返す同期的なメッセージ。

### Success Response

```json
{
  "messageType": "Success",
  "messageId": "750e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-13T10:30:01Z",
  "correlationId": "550e8400-e29b-41d4-a716-446655440000",
  "payload": {
    "success": true,
    "data": {
      "roomId": "room-123",
      "playerId": "player-001"
    }
  }
}
```

**Payload定義**:
```go
type SuccessPayload struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data,omitempty"`
}
```

### Error Response

```json
{
  "messageType": "Error",
  "messageId": "750e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2025-11-13T10:31:02Z",
  "correlationId": "550e8400-e29b-41d4-a716-446655440001",
  "payload": {
    "success": false,
    "error": {
      "code": "ROOM_FULL",
      "message": "The room is full",
      "details": {
        "roomId": "room-123",
        "maxPlayers": 4,
        "currentPlayers": 4
      }
    }
  }
}
```

**Payload定義**:
```go
type ErrorPayload struct {
    Success bool       `json:"success"`
    Error   ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### エラーコード一覧

| コード | 説明 | HTTPステータス相当 |
|--------|------|-------------------|
| `INVALID_REQUEST` | リクエストの形式が不正 | 400 |
| `VALIDATION_ERROR` | バリデーションエラー | 400 |
| `ROOM_NOT_FOUND` | ルームが見つからない | 404 |
| `ROOM_FULL` | ルームが満員 | 409 |
| `ROOM_ALREADY_EXISTS` | ルームが既に存在 | 409 |
| `GAME_ALREADY_STARTED` | ゲームが既に開始済み | 409 |
| `GAME_NOT_STARTED` | ゲームが未開始 | 409 |
| `NOT_YOUR_TURN` | あなたのターンではない | 403 |
| `MAX_ROLLS_EXCEEDED` | 最大ロール回数超過 | 409 |
| `CATEGORY_ALREADY_SCORED` | カテゴリーが既に使用済み | 409 |
| `UNAUTHORIZED` | 権限がない | 401 |
| `INTERNAL_ERROR` | サーバー内部エラー | 500 |

---

## プロトコル実装

### メッセージエンコーダー/デコーダー

```go
package protocol

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

type MessageEncoder struct{}

func NewMessageEncoder() *MessageEncoder {
    return &MessageEncoder{}
}

func (e *MessageEncoder) Encode(msgType MessageType, payload interface{}, correlationID string) ([]byte, error) {
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }

    msg := Message{
        MessageType:   string(msgType),
        MessageID:     uuid.New().String(),
        Timestamp:     time.Now().UTC().Format(time.RFC3339),
        CorrelationID: correlationID,
        Payload:       payloadBytes,
    }

    return json.Marshal(msg)
}

type MessageDecoder struct{}

func NewMessageDecoder() *MessageDecoder {
    return &MessageDecoder{}
}

func (d *MessageDecoder) Decode(data []byte) (*Message, error) {
    var msg Message
    if err := json.Unmarshal(data, &msg); err != nil {
        return nil, err
    }
    return &msg, nil
}

func (d *MessageDecoder) DecodePayload(msg *Message, v interface{}) error {
    return json.Unmarshal(msg.Payload, v)
}
```

### メッセージハンドラー

```go
package handler

type MessageHandler interface {
    HandleMessage(conn Connection, msg *protocol.Message) error
}

type messageRouter struct {
    handlers map[protocol.MessageType]CommandHandler
}

func (r *messageRouter) HandleMessage(conn Connection, msg *protocol.Message) error {
    handler, exists := r.handlers[protocol.MessageType(msg.MessageType)]
    if !exists {
        return r.sendError(conn, msg.MessageID, "UNKNOWN_MESSAGE_TYPE", "Unknown message type")
    }

    return handler.Handle(conn, msg)
}

type CommandHandler interface {
    Handle(conn Connection, msg *protocol.Message) error
}

// 具体的なハンドラー例
type CreateRoomHandler struct {
    createRoomUC *usecase.CreateRoomUseCase
    encoder      *protocol.MessageEncoder
}

func (h *CreateRoomHandler) Handle(conn Connection, msg *protocol.Message) error {
    var payload protocol.CreateRoomPayload
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        return h.sendError(conn, msg.MessageID, "INVALID_REQUEST", err.Error())
    }

    // バリデーション
    if err := validate.Struct(payload); err != nil {
        return h.sendValidationError(conn, msg.MessageID, err)
    }

    // ユースケース実行
    output, err := h.createRoomUC.Execute(context.Background(), usecase.CreateRoomInput{
        HostPlayerName: payload.PlayerName,
        MaxPlayers:     payload.MaxPlayers,
    })
    if err != nil {
        return h.sendDomainError(conn, msg.MessageID, err)
    }

    // 成功レスポンス送信
    return h.sendSuccess(conn, msg.MessageID, map[string]interface{}{
        "roomId":   output.RoomID,
        "playerId": output.PlayerID,
    })
}

func (h *CreateRoomHandler) sendSuccess(conn Connection, correlationID string, data map[string]interface{}) error {
    response, _ := h.encoder.Encode(protocol.MessageTypeSuccess, protocol.SuccessPayload{
        Success: true,
        Data:    data,
    }, correlationID)

    return conn.Send(response)
}

func (h *CreateRoomHandler) sendError(conn Connection, correlationID, code, message string) error {
    response, _ := h.encoder.Encode(protocol.MessageTypeError, protocol.ErrorPayload{
        Success: false,
        Error: protocol.ErrorInfo{
            Code:    code,
            Message: message,
        },
    }, correlationID)

    return conn.Send(response)
}
```

---

## 接続管理

### WebSocketへの拡張性

```go
// 将来的なWebSocket対応を考慮したインターフェース
type Connection interface {
    Send(data []byte) error
    Receive() ([]byte, error)
    Close() error
    RemoteAddr() string
    SetReadDeadline(t time.Time) error
    SetWriteDeadline(t time.Time) error
}

// TCP実装
type TCPConnection struct {
    conn net.Conn
}

// 将来のWebSocket実装
type WebSocketConnection struct {
    conn *websocket.Conn
}
```

### ハートビート（Keepalive）

```json
{
  "messageType": "Ping",
  "messageId": "850e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-13T10:40:00Z",
  "payload": {}
}
```

```json
{
  "messageType": "Pong",
  "messageId": "850e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2025-11-13T10:40:00Z",
  "correlationId": "850e8400-e29b-41d4-a716-446655440000",
  "payload": {}
}
```

---

## バージョニング

### プロトコルバージョン管理

```json
{
  "messageType": "CreateRoomCommand",
  "messageId": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-13T10:30:00Z",
  "protocolVersion": "1.0",
  "payload": { /* ... */ }
}
```

### 下位互換性の保証

- 新しいフィールドは常にオプショナル
- 既存のフィールド名は変更しない
- 削除予定のフィールドは `deprecated` フラグで明示

---

## まとめ

このメッセージプロトコル設計により：

✅ **可読性**: JSONによる人間にも機械にも理解しやすい形式
✅ **拡張性**: 新しいメッセージタイプの追加が容易
✅ **デバッグ性**: メッセージIDとタイムスタンプによる追跡
✅ **型安全性**: 明確なPayload定義とバリデーション
✅ **エラーハンドリング**: 標準化されたエラーレスポンス
✅ **将来性**: WebSocket等への拡張が可能

次のステップ: パッケージ構造設計書でディレクトリ構成を定義します。
