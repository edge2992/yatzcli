# レイヤーアーキテクチャ設計書

## 概要

本ドキュメントはYatzCLIのレイヤーアーキテクチャを定義します。ドメイン駆動設計の原則に従い、ビジネスロジックを中心とした同心円状のレイヤー構造を採用します。

## アーキテクチャ原則

### 1. 依存関係の方向

```
┌─────────────────────────────────────────┐
│    Presentation Layer (CLI/UI)         │
├─────────────────────────────────────────┤
│    Application Layer (Use Cases)       │
├─────────────────────────────────────────┤
│    Domain Layer (Business Logic)       │
├─────────────────────────────────────────┤
│    Infrastructure Layer (DB/Network)   │
└─────────────────────────────────────────┘

依存関係の方向: 外側 → 内側
ドメイン層は他のいかなる層にも依存しない
```

### 2. Dependency Inversion Principle（依存関係逆転の原則）

```
┌──────────────┐          ┌──────────────┐
│ Application  │────────▶ │   Domain     │
│   Layer      │          │ (Interface)  │
└──────────────┘          └──────────────┘
                                  △
                                  │
                          ┌───────┴────────┐
                          │ Infrastructure │
                          │ (Implementation)│
                          └────────────────┘
```

高レベルのモジュールは低レベルのモジュールに依存せず、両者は抽象（インターフェース）に依存する。

---

## レイヤー構成

### 1. ドメイン層（Domain Layer）

**責務**: ビジネスロジックとゲームルールの実装

**含まれるもの**:
- エンティティ（Room, Player, ScoreCard）
- 値オブジェクト（Dice, Score, GameState）
- 集約（Room集約）
- ドメインサービス（ScoreCalculationService）
- リポジトリインターフェース
- ドメインイベント
- ドメインエラー

**依存関係**: なし（標準ライブラリのみ）

**パッケージ構成**:
```
domain/
├── entity/
│   ├── room.go
│   ├── player.go
│   └── scorecard.go
├── valueobject/
│   ├── dice.go
│   ├── score.go
│   ├── ids.go
│   └── gamestate.go
├── service/
│   ├── score_calculation.go
│   ├── game_rule_validator.go
│   └── turn_manager.go
├── repository/
│   ├── room_repository.go
│   └── player_repository.go
├── event/
│   └── events.go
└── error/
    └── errors.go
```

**例**:
```go
package domain

// ドメインサービス
type ScoreCalculationService interface {
    CalculateScore(category ScoreCategory, diceSet DiceSet) Score
}

// リポジトリインターフェース（実装はインフラ層）
type RoomRepository interface {
    Save(room *Room) error
    FindByID(id RoomID) (*Room, error)
}
```

---

### 2. アプリケーション層（Application Layer）

**責務**: ユースケースの実装とトランザクション管理

**含まれるもの**:
- ユースケース（Use Case）
- コマンド（Command）
- クエリ（Query）
- DTO（Data Transfer Object）
- アプリケーションサービス

**依存関係**: ドメイン層のみ

**パッケージ構成**:
```
application/
├── usecase/
│   ├── room/
│   │   ├── create_room.go
│   │   ├── join_room.go
│   │   └── leave_room.go
│   └── game/
│       ├── start_game.go
│       ├── roll_dice.go
│       ├── choose_category.go
│       └── end_game.go
├── command/
│   ├── create_room_command.go
│   ├── join_room_command.go
│   └── roll_dice_command.go
├── query/
│   ├── get_room_query.go
│   └── list_rooms_query.go
└── dto/
    ├── room_dto.go
    └── player_dto.go
```

#### ユースケース設計

**基本構造**:
```go
package usecase

type UseCase[TInput any, TOutput any] interface {
    Execute(ctx context.Context, input TInput) (TOutput, error)
}
```

**具体例: ルーム作成ユースケース**
```go
package room

type CreateRoomUseCase struct {
    roomRepo   domain.RoomRepository
    validator  domain.GameRuleValidator
    eventBus   EventBus
}

type CreateRoomInput struct {
    HostPlayerID   domain.PlayerID
    HostPlayerName domain.PlayerName
    MaxPlayers     int
}

type CreateRoomOutput struct {
    RoomID    domain.RoomID
    CreatedAt time.Time
}

func (uc *CreateRoomUseCase) Execute(
    ctx context.Context,
    input CreateRoomInput,
) (CreateRoomOutput, error) {
    // 1. ドメインオブジェクト生成
    room := domain.NewRoom(domain.NewRoomID())
    room.SetMaxPlayers(input.MaxPlayers)

    host := domain.NewPlayer(input.HostPlayerID, input.HostPlayerName)

    // 2. ビジネスロジック実行
    if err := room.AddPlayer(host); err != nil {
        return CreateRoomOutput{}, err
    }

    // 3. 永続化
    if err := uc.roomRepo.Save(room); err != nil {
        return CreateRoomOutput{}, err
    }

    // 4. イベント発行
    uc.eventBus.Publish(domain.RoomCreatedEvent{
        RoomID:    room.ID(),
        HostID:    host.ID(),
        Timestamp: time.Now(),
    })

    return CreateRoomOutput{
        RoomID:    room.ID(),
        CreatedAt: room.CreatedAt(),
    }, nil
}
```

**具体例: ダイスロールユースケース**
```go
package game

type RollDiceUseCase struct {
    roomRepo      domain.RoomRepository
    turnManager   domain.TurnManager
    validator     domain.GameRuleValidator
}

type RollDiceInput struct {
    RoomID     domain.RoomID
    PlayerID   domain.PlayerID
    HeldIndices []int
}

type RollDiceOutput struct {
    DiceSet   domain.DiceSet
    RollCount int
    CanRoll   bool
}

func (uc *RollDiceUseCase) Execute(
    ctx context.Context,
    input RollDiceInput,
) (RollDiceOutput, error) {
    // 1. 集約取得
    room, err := uc.roomRepo.FindByID(input.RoomID)
    if err != nil {
        return RollDiceOutput{}, err
    }

    // 2. バリデーション
    if err := uc.validator.CanRollDice(room, input.PlayerID); err != nil {
        return RollDiceOutput{}, err
    }

    // 3. ドメインロジック実行
    turn := room.GetCurrentTurn()
    if err := turn.Roll(); err != nil {
        return RollDiceOutput{}, err
    }

    // 4. 永続化
    if err := uc.roomRepo.Save(room); err != nil {
        return RollDiceOutput{}, err
    }

    return RollDiceOutput{
        DiceSet:   turn.GetDiceSet(),
        RollCount: turn.RollCount(),
        CanRoll:   turn.CanRoll(),
    }, nil
}
```

#### Command Query Responsibility Segregation (CQRS)

**コマンド（書き込み操作）**:
```go
type Command interface {
    CommandName() string
}

type CreateRoomCommand struct {
    HostPlayerID   string
    HostPlayerName string
    MaxPlayers     int
}

type CommandHandler[T Command] interface {
    Handle(ctx context.Context, cmd T) error
}
```

**クエリ（読み取り操作）**:
```go
type Query[TResult any] interface {
    QueryName() string
}

type GetRoomQuery struct {
    RoomID string
}

type RoomDetailDTO struct {
    RoomID      string
    Players     []PlayerDTO
    GameState   string
    CurrentTurn int
}

type QueryHandler[TQuery Query[TResult], TResult any] interface {
    Handle(ctx context.Context, query TQuery) (TResult, error)
}
```

---

### 3. インフラ層（Infrastructure Layer）

**責務**: 外部システムとの統合と技術的実装

**含まれるもの**:
- リポジトリ実装
- ネットワーク通信
- データベース接続
- ファイルI/O
- ロギング
- キャッシング

**依存関係**: ドメイン層、アプリケーション層

**パッケージ構成**:
```
infrastructure/
├── persistence/
│   ├── memory/
│   │   ├── room_repository.go
│   │   └── player_repository.go
│   └── redis/
│       └── room_repository.go
├── network/
│   ├── server/
│   │   ├── tcp_server.go
│   │   └── connection_manager.go
│   └── protocol/
│       ├── json_protocol.go
│       └── message_codec.go
├── messaging/
│   ├── event_bus.go
│   └── message_broker.go
└── logging/
    └── logger.go
```

#### リポジトリ実装例

**インメモリ実装**:
```go
package memory

type RoomRepository struct {
    rooms map[domain.RoomID]*domain.Room
    mu    sync.RWMutex
}

func NewRoomRepository() *RoomRepository {
    return &RoomRepository{
        rooms: make(map[domain.RoomID]*domain.Room),
    }
}

func (r *RoomRepository) Save(room *domain.Room) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.rooms[room.ID()] = room
    return nil
}

func (r *RoomRepository) FindByID(id domain.RoomID) (*domain.Room, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    room, exists := r.rooms[id]
    if !exists {
        return nil, domain.ErrRoomNotFound
    }
    return room, nil
}
```

#### ネットワーク通信

**TCP Server**:
```go
package server

type TCPServer struct {
    address    string
    listener   net.Listener
    connMgr    *ConnectionManager
    handler    MessageHandler
    logger     Logger
}

func (s *TCPServer) Start() error {
    listener, err := net.Listen("tcp", s.address)
    if err != nil {
        return err
    }
    s.listener = listener

    for {
        conn, err := listener.Accept()
        if err != nil {
            s.logger.Error("accept error", err)
            continue
        }

        go s.handleConnection(conn)
    }
}

func (s *TCPServer) handleConnection(conn net.Conn) {
    defer conn.Close()

    connection := NewConnection(conn)
    s.connMgr.Register(connection)
    defer s.connMgr.Unregister(connection)

    for {
        msg, err := connection.ReadMessage()
        if err != nil {
            return
        }

        s.handler.Handle(connection, msg)
    }
}
```

**メッセージコーデック**:
```go
package protocol

type MessageCodec interface {
    Encode(message Message) ([]byte, error)
    Decode(data []byte) (Message, error)
}

type JSONCodec struct{}

func (c *JSONCodec) Encode(message Message) ([]byte, error) {
    return json.Marshal(message)
}

func (c *JSONCodec) Decode(data []byte) (Message, error) {
    var msg Message
    err := json.Unmarshal(data, &msg)
    return msg, err
}
```

---

### 4. プレゼンテーション層（Presentation Layer）

**責務**: ユーザーインターフェースとユーザー入力の処理

**含まれるもの**:
- CLI ハンドラー
- 入力バリデーション
- 出力フォーマット
- ユーザー通知

**依存関係**: アプリケーション層

**パッケージ構成**:
```
presentation/
├── cli/
│   ├── command/
│   │   ├── create_room_command.go
│   │   ├── join_room_command.go
│   │   └── play_game_command.go
│   ├── view/
│   │   ├── room_view.go
│   │   ├── game_view.go
│   │   └── scoreboard_view.go
│   └── input/
│       ├── player_input.go
│       └── validation.go
└── server/
    ├── handler/
    │   ├── message_handler.go
    │   └── connection_handler.go
    └── router/
        └── message_router.go
```

#### CLIハンドラー例

```go
package cli

type GameCLI struct {
    rollDiceUC     *usecase.RollDiceUseCase
    chooseCatUC    *usecase.ChooseCategoryUseCase
    inputHandler   InputHandler
    gameView       *view.GameView
}

func (cli *GameCLI) PlayTurn(roomID string, playerID string) error {
    // 1. ユーザー入力取得
    heldIndices, err := cli.inputHandler.GetDiceHoldInput()
    if err != nil {
        return err
    }

    // 2. ユースケース実行
    output, err := cli.rollDiceUC.Execute(context.Background(), usecase.RollDiceInput{
        RoomID:      domain.RoomID(roomID),
        PlayerID:    domain.PlayerID(playerID),
        HeldIndices: heldIndices,
    })
    if err != nil {
        return cli.handleError(err)
    }

    // 3. 結果表示
    cli.gameView.DisplayDice(output.DiceSet)
    cli.gameView.DisplayRollCount(output.RollCount)

    return nil
}

func (cli *GameCLI) handleError(err error) error {
    switch {
    case errors.Is(err, domain.ErrMaxRollsExceeded):
        fmt.Println("最大ロール回数に達しました。カテゴリーを選択してください。")
    case errors.Is(err, domain.ErrNotPlayerTurn):
        fmt.Println("あなたのターンではありません。")
    default:
        fmt.Printf("エラーが発生しました: %v\n", err)
    }
    return err
}
```

#### サーバーハンドラー例

```go
package handler

type MessageHandler struct {
    createRoomUC  *usecase.CreateRoomUseCase
    joinRoomUC    *usecase.JoinRoomUseCase
    rollDiceUC    *usecase.RollDiceUseCase
    connMgr       *ConnectionManager
}

func (h *MessageHandler) Handle(conn Connection, msg Message) {
    switch msg.Type {
    case "CreateRoom":
        h.handleCreateRoom(conn, msg)
    case "JoinRoom":
        h.handleJoinRoom(conn, msg)
    case "RollDice":
        h.handleRollDice(conn, msg)
    default:
        conn.SendError("Unknown message type")
    }
}

func (h *MessageHandler) handleRollDice(conn Connection, msg Message) {
    var req RollDiceRequest
    if err := msg.UnmarshalPayload(&req); err != nil {
        conn.SendError("Invalid request format")
        return
    }

    output, err := h.rollDiceUC.Execute(context.Background(), usecase.RollDiceInput{
        RoomID:      domain.RoomID(req.RoomID),
        PlayerID:    domain.PlayerID(conn.PlayerID()),
        HeldIndices: req.HeldIndices,
    })
    if err != nil {
        conn.SendError(err.Error())
        return
    }

    // 全プレイヤーにブロードキャスト
    h.broadcastToRoom(req.RoomID, DiceRolledEvent{
        PlayerID:  conn.PlayerID(),
        DiceSet:   output.DiceSet,
        RollCount: output.RollCount,
    })
}
```

---

## レイヤー間の通信フロー

### 標準的なリクエストフロー

```
┌──────────────┐
│    User      │
└──────┬───────┘
       │ CLI入力
       ▼
┌──────────────────────────┐
│  Presentation Layer      │
│  - 入力バリデーション      │
│  - DTOへの変換            │
└──────┬───────────────────┘
       │ DTO/Command
       ▼
┌──────────────────────────┐
│  Application Layer       │
│  - UseCase実行           │
│  - トランザクション管理    │
└──────┬───────────────────┘
       │ ドメインオブジェクト操作
       ▼
┌──────────────────────────┐
│  Domain Layer            │
│  - ビジネスロジック       │
│  - 不変条件チェック       │
└──────┬───────────────────┘
       │ Repository Interface
       ▼
┌──────────────────────────┐
│  Infrastructure Layer    │
│  - 永続化                │
│  - ネットワーク通信       │
└──────────────────────────┘
```

### 具体例: ダイスロールのフロー

```go
// 1. Presentation Layer: ユーザー入力処理
func (cli *GameCLI) PlayTurn(roomID, playerID string) error {
    heldIndices := cli.inputHandler.GetDiceHoldInput()

    // 2. Application Layer: ユースケース呼び出し
    output, err := cli.rollDiceUC.Execute(ctx, usecase.RollDiceInput{
        RoomID:      domain.RoomID(roomID),
        PlayerID:    domain.PlayerID(playerID),
        HeldIndices: heldIndices,
    })
    if err != nil {
        return err
    }

    // 5. Presentation Layer: 結果表示
    cli.gameView.DisplayDice(output.DiceSet)
    return nil
}

// Application Layer: ユースケース実装
func (uc *RollDiceUseCase) Execute(ctx context.Context, input RollDiceInput) (RollDiceOutput, error) {
    // 3. Domain Layer: 集約取得とビジネスロジック実行
    room, _ := uc.roomRepo.FindByID(input.RoomID)
    turn := room.GetCurrentTurn()
    turn.Roll()

    // 4. Infrastructure Layer: 永続化
    uc.roomRepo.Save(room)

    return RollDiceOutput{DiceSet: turn.GetDiceSet()}, nil
}
```

---

## 依存性注入（Dependency Injection）

### コンストラクタインジェクション

```go
package main

func main() {
    // Infrastructure Layer
    roomRepo := memory.NewRoomRepository()
    playerRepo := memory.NewPlayerRepository()

    // Domain Services
    scoreCalc := domain.NewScoreCalculationService()
    validator := domain.NewGameRuleValidator()
    turnMgr := domain.NewTurnManager()

    // Application Layer (Use Cases)
    createRoomUC := usecase.NewCreateRoomUseCase(roomRepo, validator)
    joinRoomUC := usecase.NewJoinRoomUseCase(roomRepo, playerRepo, validator)
    rollDiceUC := usecase.NewRollDiceUseCase(roomRepo, turnMgr, validator)
    chooseCategoryUC := usecase.NewChooseCategoryUseCase(roomRepo, scoreCalc, validator)

    // Presentation Layer
    gameCLI := cli.NewGameCLI(
        rollDiceUC,
        chooseCategoryUC,
        cli.NewInputHandler(),
        view.NewGameView(),
    )

    // Server
    messageHandler := handler.NewMessageHandler(
        createRoomUC,
        joinRoomUC,
        rollDiceUC,
        chooseCategoryUC,
    )

    server := server.NewTCPServer(":8080", messageHandler)
    server.Start()
}
```

### インターフェースによる抽象化

```go
// ドメイン層でインターフェース定義
type RoomRepository interface {
    Save(room *Room) error
    FindByID(id RoomID) (*Room, error)
}

// インフラ層で実装
type InMemoryRoomRepository struct { /* ... */ }
type RedisRoomRepository struct { /* ... */ }

// アプリケーション層はインターフェースに依存
type CreateRoomUseCase struct {
    roomRepo domain.RoomRepository // インターフェース
}
```

---

## テスタビリティ

各レイヤーを独立してテスト可能：

```go
// ドメイン層のテスト（依存なし）
func TestRoom_AddPlayer(t *testing.T) {
    room := domain.NewRoom(domain.NewRoomID())
    player := domain.NewPlayer(domain.NewPlayerID(), domain.NewPlayerName("Alice"))

    err := room.AddPlayer(player)
    assert.NoError(t, err)
}

// アプリケーション層のテスト（モックリポジトリ使用）
func TestCreateRoomUseCase_Execute(t *testing.T) {
    mockRepo := &MockRoomRepository{}
    uc := usecase.NewCreateRoomUseCase(mockRepo, validator)

    output, err := uc.Execute(ctx, usecase.CreateRoomInput{
        HostPlayerID: "player1",
    })

    assert.NoError(t, err)
    assert.True(t, mockRepo.SaveCalled)
}

// プレゼンテーション層のテスト（モックユースケース使用）
func TestGameCLI_PlayTurn(t *testing.T) {
    mockUC := &MockRollDiceUseCase{
        Output: usecase.RollDiceOutput{DiceSet: testDiceSet},
    }
    cli := cli.NewGameCLI(mockUC, ...)

    err := cli.PlayTurn("room1", "player1")
    assert.NoError(t, err)
}
```

---

## まとめ

このレイヤーアーキテクチャにより：

✅ **明確な責務分離**: 各レイヤーの役割が明確
✅ **技術的独立性**: ドメインロジックは技術詳細から独立
✅ **テスタビリティ**: 各レイヤーを独立してテスト可能
✅ **拡張性**: 新機能の追加が容易
✅ **保守性**: 変更の影響範囲が限定的
✅ **依存性逆転**: インターフェースによる柔軟な実装切り替え

次のステップ: 状態管理設計書でゲーム状態の遷移とターン管理を詳細化します。
