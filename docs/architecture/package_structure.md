# パッケージ構造設計書

## 概要

本ドキュメントはYatzCLIの理想的なパッケージ構造を定義します。ドメイン駆動設計のレイヤーアーキテクチャに基づき、明確な責務分離と依存関係の方向性を保証します。

## プロジェクトルート構成

```
yatzcli/
├── cmd/                    # エントリーポイント
│   ├── server/            # サーバーアプリケーション
│   └── client/            # クライアントアプリケーション
├── internal/              # 内部パッケージ（外部から非公開）
│   ├── domain/           # ドメイン層
│   ├── application/      # アプリケーション層
│   ├── infrastructure/   # インフラ層
│   └── presentation/     # プレゼンテーション層
├── pkg/                   # 公開可能なパッケージ（オプション）
│   └── protocol/         # メッセージプロトコル定義
├── docs/                  # ドキュメント
├── test/                  # 統合テスト・E2Eテスト
├── scripts/               # ビルド・デプロイスクリプト
├── configs/               # 設定ファイル
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## cmd/ - エントリーポイント

各アプリケーションのメインパッケージ。依存性の組み立てと起動を行います。

```
cmd/
├── server/
│   └── main.go           # サーバー起動
└── client/
    └── main.go           # クライアント起動
```

### cmd/server/main.go

```go
package main

import (
    "log"
    "os"

    "github.com/edge2992/yatzcli/internal/application/usecase"
    "github.com/edge2992/yatzcli/internal/domain/service"
    "github.com/edge2992/yatzcli/internal/infrastructure/persistence/memory"
    "github.com/edge2992/yatzcli/internal/infrastructure/server"
    "github.com/edge2992/yatzcli/internal/presentation/handler"
)

func main() {
    // 設定読み込み
    config := loadConfig()

    // Infrastructure Layer
    roomRepo := memory.NewRoomRepository()
    playerRepo := memory.NewPlayerRepository()
    eventBus := memory.NewEventBus()

    // Domain Services
    scoreCalc := service.NewScoreCalculationService()
    validator := service.NewGameRuleValidator()
    turnMgr := service.NewTurnManager()

    // Application Layer (Use Cases)
    createRoomUC := usecase.NewCreateRoomUseCase(roomRepo, validator, eventBus)
    joinRoomUC := usecase.NewJoinRoomUseCase(roomRepo, playerRepo, validator, eventBus)
    startGameUC := usecase.NewStartGameUseCase(roomRepo, validator, eventBus)
    rollDiceUC := usecase.NewRollDiceUseCase(roomRepo, turnMgr, validator, eventBus)
    chooseCategoryUC := usecase.NewChooseCategoryUseCase(roomRepo, scoreCalc, validator, eventBus)

    // Presentation Layer
    messageHandler := handler.NewMessageRouter(
        createRoomUC,
        joinRoomUC,
        startGameUC,
        rollDiceUC,
        chooseCategoryUC,
    )

    // Server
    srv := server.NewTCPServer(config.Address, messageHandler)

    log.Printf("Server starting on %s", config.Address)
    if err := srv.Start(); err != nil {
        log.Fatal(err)
    }
}

func loadConfig() *Config {
    return &Config{
        Address: getEnv("SERVER_ADDRESS", ":8080"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

---

## internal/domain/ - ドメイン層

ビジネスロジックとゲームルールの中核。他のパッケージに依存しません。

```
internal/domain/
├── entity/               # エンティティ
│   ├── room.go
│   ├── room_test.go
│   ├── player.go
│   ├── player_test.go
│   ├── scorecard.go
│   └── scorecard_test.go
├── valueobject/         # 値オブジェクト
│   ├── dice.go
│   ├── dice_test.go
│   ├── score.go
│   ├── turn.go
│   ├── turn_test.go
│   ├── ids.go
│   └── states.go
├── service/             # ドメインサービス
│   ├── score_calculation.go
│   ├── score_calculation_test.go
│   ├── game_rule_validator.go
│   ├── game_rule_validator_test.go
│   ├── turn_manager.go
│   └── turn_manager_test.go
├── repository/          # リポジトリインターフェース
│   ├── room_repository.go
│   └── player_repository.go
├── event/               # ドメインイベント
│   ├── event.go
│   ├── room_events.go
│   ├── game_events.go
│   └── event_bus.go
└── error/               # ドメインエラー
    └── errors.go
```

### パッケージ責務

**entity/**
- Room、Player、ScoreCardなどの識別子を持つオブジェクト
- 集約ルートと集約内エンティティ
- ビジネスロジックと不変条件の保護

**valueobject/**
- Dice、Score、Turnなどの不変オブジェクト
- 値による等価性判定
- ビジネスルールの表現

**service/**
- エンティティや値オブジェクトに所属しない振る舞い
- 複数のエンティティを横断するロジック
- ステートレスなサービス

**repository/**
- 永続化のインターフェース定義
- 実装はインフラ層が提供

**event/**
- ドメインイベントの定義
- イベントバスインターフェース

**error/**
- ドメイン固有のエラー型

---

## internal/application/ - アプリケーション層

ユースケースの実装とオーケストレーション。

```
internal/application/
├── usecase/
│   ├── room/
│   │   ├── create_room.go
│   │   ├── create_room_test.go
│   │   ├── join_room.go
│   │   ├── join_room_test.go
│   │   ├── leave_room.go
│   │   └── list_rooms.go
│   └── game/
│       ├── start_game.go
│       ├── start_game_test.go
│       ├── roll_dice.go
│       ├── roll_dice_test.go
│       ├── choose_category.go
│       ├── choose_category_test.go
│       └── end_game.go
├── command/             # コマンド定義（CQRS）
│   ├── command.go
│   ├── create_room_command.go
│   ├── join_room_command.go
│   └── roll_dice_command.go
├── query/               # クエリ定義（CQRS）
│   ├── query.go
│   ├── get_room_query.go
│   └── list_rooms_query.go
├── dto/                 # データ転送オブジェクト
│   ├── room_dto.go
│   ├── player_dto.go
│   └── game_dto.go
└── service/             # アプリケーションサービス
    └── transaction.go   # トランザクション管理
```

### ユースケース例

**usecase/room/create_room.go**:
```go
package room

import (
    "context"

    "github.com/edge2992/yatzcli/internal/domain/entity"
    "github.com/edge2992/yatzcli/internal/domain/event"
    "github.com/edge2992/yatzcli/internal/domain/repository"
    "github.com/edge2992/yatzcli/internal/domain/service"
    "github.com/edge2992/yatzcli/internal/domain/valueobject"
)

type CreateRoomUseCase struct {
    roomRepo  repository.RoomRepository
    validator service.GameRuleValidator
    eventBus  event.EventBus
}

type CreateRoomInput struct {
    HostPlayerID   valueobject.PlayerID
    HostPlayerName valueobject.PlayerName
    MaxPlayers     int
}

type CreateRoomOutput struct {
    RoomID    valueobject.RoomID
    PlayerID  valueobject.PlayerID
    CreatedAt time.Time
}

func (uc *CreateRoomUseCase) Execute(
    ctx context.Context,
    input CreateRoomInput,
) (CreateRoomOutput, error) {
    // ドメインオブジェクト生成
    room := entity.NewRoom(valueobject.NewRoomID())
    room.SetMaxPlayers(input.MaxPlayers)

    host := entity.NewPlayer(input.HostPlayerID, input.HostPlayerName)

    // ビジネスロジック実行
    if err := room.AddPlayer(host); err != nil {
        return CreateRoomOutput{}, err
    }

    // 永続化
    if err := uc.roomRepo.Save(room); err != nil {
        return CreateRoomOutput{}, err
    }

    // イベント発行
    uc.eventBus.Publish(event.RoomCreatedEvent{
        RoomID:    room.ID(),
        HostID:    host.ID(),
        Timestamp: time.Now(),
    })

    return CreateRoomOutput{
        RoomID:    room.ID(),
        PlayerID:  host.ID(),
        CreatedAt: room.CreatedAt(),
    }, nil
}
```

---

## internal/infrastructure/ - インフラ層

技術的実装と外部システムとの統合。

```
internal/infrastructure/
├── persistence/
│   ├── memory/          # インメモリ実装
│   │   ├── room_repository.go
│   │   ├── room_repository_test.go
│   │   ├── player_repository.go
│   │   └── event_bus.go
│   └── redis/           # Redis実装（将来）
│       └── room_repository.go
├── server/              # サーバー実装
│   ├── tcp_server.go
│   ├── tcp_server_test.go
│   ├── connection.go
│   ├── connection_manager.go
│   └── connection_manager_test.go
├── client/              # クライアント接続
│   ├── tcp_client.go
│   └── connection.go
├── messaging/           # メッセージング
│   ├── event_bus.go
│   └── message_broker.go
├── logging/             # ロギング
│   ├── logger.go
│   └── zap_logger.go
└── config/              # 設定管理
    ├── config.go
    └── loader.go
```

### リポジトリ実装例

**persistence/memory/room_repository.go**:
```go
package memory

import (
    "sync"

    "github.com/edge2992/yatzcli/internal/domain/entity"
    "github.com/edge2992/yatzcli/internal/domain/repository"
    "github.com/edge2992/yatzcli/internal/domain/valueobject"
)

type RoomRepository struct {
    rooms map[valueobject.RoomID]*entity.Room
    mu    sync.RWMutex
}

func NewRoomRepository() repository.RoomRepository {
    return &RoomRepository{
        rooms: make(map[valueobject.RoomID]*entity.Room),
    }
}

func (r *RoomRepository) Save(room *entity.Room) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.rooms[room.ID()] = room
    return nil
}

func (r *RoomRepository) FindByID(id valueobject.RoomID) (*entity.Room, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    room, exists := r.rooms[id]
    if !exists {
        return nil, domain.ErrRoomNotFound
    }
    return room, nil
}
```

---

## internal/presentation/ - プレゼンテーション層

ユーザーインターフェースとメッセージハンドリング。

```
internal/presentation/
├── handler/             # サーバーサイドハンドラー
│   ├── message_router.go
│   ├── message_router_test.go
│   ├── create_room_handler.go
│   ├── join_room_handler.go
│   ├── roll_dice_handler.go
│   └── error_handler.go
├── cli/                 # CLIクライアント
│   ├── game_cli.go
│   ├── game_cli_test.go
│   ├── input/
│   │   ├── input_handler.go
│   │   ├── survey_input.go
│   │   └── validation.go
│   └── view/
│       ├── room_view.go
│       ├── game_view.go
│       └── scoreboard_view.go
└── middleware/          # ミドルウェア
    ├── auth.go
    ├── logging.go
    └── rate_limit.go
```

### ハンドラー例

**handler/create_room_handler.go**:
```go
package handler

import (
    "context"
    "encoding/json"

    "github.com/edge2992/yatzcli/internal/application/usecase/room"
    "github.com/edge2992/yatzcli/pkg/protocol"
)

type CreateRoomHandler struct {
    createRoomUC *room.CreateRoomUseCase
    encoder      *protocol.MessageEncoder
}

func NewCreateRoomHandler(uc *room.CreateRoomUseCase) *CreateRoomHandler {
    return &CreateRoomHandler{
        createRoomUC: uc,
        encoder:      protocol.NewMessageEncoder(),
    }
}

func (h *CreateRoomHandler) Handle(conn Connection, msg *protocol.Message) error {
    var payload protocol.CreateRoomPayload
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        return h.sendError(conn, msg.MessageID, "INVALID_REQUEST", err.Error())
    }

    output, err := h.createRoomUC.Execute(context.Background(), room.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName(payload.PlayerName),
        MaxPlayers:     payload.MaxPlayers,
    })
    if err != nil {
        return h.sendDomainError(conn, msg.MessageID, err)
    }

    return h.sendSuccess(conn, msg.MessageID, map[string]interface{}{
        "roomId":   string(output.RoomID),
        "playerId": string(output.PlayerID),
    })
}
```

---

## pkg/ - 公開パッケージ

外部から利用可能なパッケージ（オプション）。

```
pkg/
└── protocol/            # メッセージプロトコル定義
    ├── message.go
    ├── commands.go
    ├── events.go
    ├── encoder.go
    └── decoder.go
```

このパッケージは将来的に他のプロジェクトから利用される可能性がある場合に配置します。

---

## test/ - 統合テスト

```
test/
├── integration/         # 統合テスト
│   ├── room_test.go
│   ├── game_flow_test.go
│   └── helpers.go
├── e2e/                 # E2Eテスト
│   ├── full_game_test.go
│   └── multiplayer_test.go
└── fixtures/            # テストデータ
    └── test_data.go
```

---

## configs/ - 設定ファイル

```
configs/
├── server.yaml          # サーバー設定
├── client.yaml          # クライアント設定
├── development.yaml     # 開発環境設定
└── production.yaml      # 本番環境設定
```

---

## 依存関係の可視化

### レイヤー間の依存関係

```
┌────────────────────────────────────────────┐
│              cmd/                          │
│         (main packages)                    │
└──────────┬────────────────────────────────┘
           │ imports
           ▼
┌────────────────────────────────────────────┐
│        internal/presentation/              │
│          (handlers, CLI)                   │
└──────────┬────────────────────────────────┘
           │ imports
           ▼
┌────────────────────────────────────────────┐
│       internal/application/                │
│          (use cases, DTOs)                 │
└──────────┬────────────────────────────────┘
           │ imports
           ▼
┌────────────────────────────────────────────┐
│          internal/domain/                  │
│  (entities, services, interfaces)          │
└────────────────────────────────────────────┘
           △
           │ implements
           │
┌──────────┴────────────────────────────────┐
│      internal/infrastructure/              │
│    (repositories, server, client)          │
└────────────────────────────────────────────┘
```

### import ルール

1. **domain層**: 標準ライブラリのみに依存
2. **application層**: domainのみに依存
3. **infrastructure層**: domain、applicationに依存可能
4. **presentation層**: application、infrastructureに依存可能
5. **cmd/**: すべての層に依存可能（組み立てのため）

---

## ファイル命名規則

### テストファイル

- ユニットテスト: `*_test.go`（同一パッケージ）
- モックファイル: `mock_*.go`

### 実装ファイル

- インターフェース: `*_interface.go` または `interface.go`
- 実装: 具体的な名前（例: `tcp_server.go`、`memory_repository.go`）

---

## パッケージサイズの目安

| レベル | 目安行数 | ファイル数 |
|-------|---------|-----------|
| 小 | 〜500行 | 1-3ファイル |
| 中 | 500-1500行 | 3-7ファイル |
| 大 | 1500行〜 | 7+ファイル（分割を検討） |

---

## Makefile の構成

```makefile
.PHONY: build test lint run-server run-client

# ビルド
build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/client cmd/client/main.go

# テスト
test:
	go test ./internal/... -cover

test-integration:
	go test ./test/integration/... -tags=integration

test-e2e:
	go test ./test/e2e/... -tags=e2e

# カバレッジ
coverage:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

# Lint
lint:
	golangci-lint run

# 実行
run-server:
	go run cmd/server/main.go

run-client:
	go run cmd/client/main.go

# クリーンアップ
clean:
	rm -rf bin/
	rm -f coverage.out
```

---

## go.mod の依存関係

```go
module github.com/edge2992/yatzcli

go 1.21

require (
    github.com/google/uuid v1.4.0
    github.com/stretchr/testify v1.8.4
    github.com/AlecAivazis/survey/v2 v2.3.7
    github.com/fatih/color v1.15.0
    github.com/olekukonko/tablewriter v0.0.5
    go.uber.org/zap v1.26.0          // ロギング
    github.com/go-playground/validator/v10 v10.15.5  // バリデーション
)
```

---

## まとめ

このパッケージ構造設計により：

✅ **明確な責務分離**: 各レイヤーの役割が明確
✅ **依存関係の制御**: 一方向の依存関係
✅ **テスタビリティ**: 各層を独立してテスト可能
✅ **拡張性**: 新機能の追加が容易
✅ **保守性**: コードの配置場所が予測可能
✅ **スケーラビリティ**: 大規模化に対応可能な構造

次のステップ: クラス図・シーケンス図で視覚的な設計を提供します。
