# テスタビリティ設計書

## 概要

本ドキュメントはYatzCLIのテスト戦略と実装方法を定義します。高いテスタビリティを実現するための依存性注入、モック設計、テストパターンを提供します。

## テスト原則

### 1. テストピラミッド

```
           /\
          /  \         E2E Tests (少数)
         /────\        - 完全なシナリオ
        /      \       - 複数プレイヤー
       /────────\      - 実際のネットワーク
      /          \
     / Integration \   Integration Tests (中程度)
    /──────────────\   - レイヤー間連携
   /                \  - リポジトリ実装
  /   Unit Tests    \ Unit Tests (多数)
 /──────────────────\ - ドメインロジック
/____________________\ - 値オブジェクト
                       - サービス
```

**目標比率**: Unit 70% : Integration 20% : E2E 10%

### 2. FIRST原則

- **Fast**: 高速に実行
- **Independent**: 独立して実行可能
- **Repeatable**: 何度でも同じ結果
- **Self-Validating**: 自己検証可能
- **Timely**: 適切なタイミングで作成

### 3. AAA (Arrange-Act-Assert) パターン

```go
func TestRoom_AddPlayer(t *testing.T) {
    // Arrange: テストの準備
    room := domain.NewRoom(domain.NewRoomID())
    player := domain.NewPlayer(domain.NewPlayerID(), domain.NewPlayerName("Alice"))

    // Act: 実行
    err := room.AddPlayer(player)

    // Assert: 検証
    assert.NoError(t, err)
    assert.Equal(t, 1, len(room.GetPlayers()))
}
```

---

## 依存性注入設計

### 1. コンストラクタインジェクション

```go
// ユースケース
type CreateRoomUseCase struct {
    roomRepo  domain.RoomRepository       // インターフェース
    validator domain.GameRuleValidator   // インターフェース
    eventBus  event.EventBus             // インターフェース
}

func NewCreateRoomUseCase(
    roomRepo domain.RoomRepository,
    validator domain.GameRuleValidator,
    eventBus event.EventBus,
) *CreateRoomUseCase {
    return &CreateRoomUseCase{
        roomRepo:  roomRepo,
        validator: validator,
        eventBus:  eventBus,
    }
}
```

**メリット**:
- 依存関係が明示的
- イミュータブル
- テスト時にモックを注入可能

### 2. インターフェース定義

```go
// domain/repository/room_repository.go
package repository

type RoomRepository interface {
    Save(room *entity.Room) error
    FindByID(id valueobject.RoomID) (*entity.Room, error)
    FindAll() ([]*entity.Room, error)
    Delete(id valueobject.RoomID) error
}
```

**テスト時**:
```go
// モック実装を注入
mockRepo := &MockRoomRepository{}
usecase := NewCreateRoomUseCase(mockRepo, validator, eventBus)
```

---

## モック・スタブ・フェイク

### 1. モック（Mock）

動作を検証するテストダブル。呼び出し回数やパラメータを記録。

```go
// internal/domain/repository/mock/mock_room_repository.go
package mock

type MockRoomRepository struct {
    SaveCalled       bool
    SaveCalledWith   *entity.Room
    SaveReturnError  error

    FindByIDCalled     bool
    FindByIDCalledWith valueobject.RoomID
    FindByIDReturn     *entity.Room
    FindByIDReturnErr  error
}

func (m *MockRoomRepository) Save(room *entity.Room) error {
    m.SaveCalled = true
    m.SaveCalledWith = room
    return m.SaveReturnError
}

func (m *MockRoomRepository) FindByID(id valueobject.RoomID) (*entity.Room, error) {
    m.FindByIDCalled = true
    m.FindByIDCalledWith = id
    return m.FindByIDReturn, m.FindByIDReturnErr
}
```

**使用例**:
```go
func TestCreateRoomUseCase_Execute(t *testing.T) {
    // Arrange
    mockRepo := &mock.MockRoomRepository{}
    mockValidator := &mock.MockGameRuleValidator{}
    mockEventBus := &mock.MockEventBus{}

    usecase := usecase.NewCreateRoomUseCase(mockRepo, mockValidator, mockEventBus)

    input := usecase.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName("Alice"),
        MaxPlayers:     4,
    }

    // Act
    output, err := usecase.Execute(context.Background(), input)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, output.RoomID)

    // モックの検証
    assert.True(t, mockRepo.SaveCalled)
    assert.NotNil(t, mockRepo.SaveCalledWith)
    assert.True(t, mockEventBus.PublishCalled)
}
```

### 2. スタブ（Stub）

固定値を返すシンプルなテストダブル。

```go
type StubRoomRepository struct {
    room *entity.Room
    err  error
}

func (s *StubRoomRepository) FindByID(id valueobject.RoomID) (*entity.Room, error) {
    return s.room, s.err
}

func (s *StubRoomRepository) Save(room *entity.Room) error {
    return nil
}
```

### 3. フェイク（Fake）

動作する軽量実装（インメモリなど）。

```go
// test/fake/fake_room_repository.go
package fake

type FakeRoomRepository struct {
    rooms map[valueobject.RoomID]*entity.Room
    mu    sync.RWMutex
}

func NewFakeRoomRepository() *FakeRoomRepository {
    return &FakeRoomRepository{
        rooms: make(map[valueobject.RoomID]*entity.Room),
    }
}

func (f *FakeRoomRepository) Save(room *entity.Room) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.rooms[room.ID()] = room
    return nil
}

func (f *FakeRoomRepository) FindByID(id valueobject.RoomID) (*entity.Room, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    room, exists := f.rooms[id]
    if !exists {
        return nil, domain.ErrRoomNotFound
    }
    return room, nil
}
```

**使い分け**:
- **Mock**: 振る舞いの検証が必要な場合
- **Stub**: 単純な戻り値のみ必要な場合
- **Fake**: 複雑なロジックを含む統合テストで使用

---

## テスト実装パターン

### 1. ドメイン層のユニットテスト

```go
// internal/domain/entity/room_test.go
package entity_test

import (
    "testing"

    "github.com/edge2992/yatzcli/internal/domain/entity"
    "github.com/edge2992/yatzcli/internal/domain/valueobject"
    "github.com/stretchr/testify/assert"
)

func TestRoom_AddPlayer_Success(t *testing.T) {
    // Arrange
    room := entity.NewRoom(valueobject.NewRoomID())
    player := entity.NewPlayer(
        valueobject.NewPlayerID(),
        valueobject.NewPlayerName("Alice"),
    )

    // Act
    err := room.AddPlayer(player)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 1, len(room.GetPlayers()))

    retrievedPlayer, err := room.GetPlayer(player.ID())
    assert.NoError(t, err)
    assert.Equal(t, player.ID(), retrievedPlayer.ID())
}

func TestRoom_AddPlayer_GameAlreadyStarted(t *testing.T) {
    // Arrange
    room := entity.NewRoom(valueobject.NewRoomID())
    player1 := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Alice"))
    player2 := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Bob"))

    room.AddPlayer(player1)
    room.AddPlayer(player2)
    room.StartGame() // ゲーム開始

    newPlayer := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Charlie"))

    // Act
    err := room.AddPlayer(newPlayer)

    // Assert
    assert.Error(t, err)
    assert.Equal(t, domain.ErrGameAlreadyStarted, err)
}

func TestRoom_AddPlayer_RoomFull(t *testing.T) {
    // Arrange
    room := entity.NewRoom(valueobject.NewRoomID())
    room.SetMaxPlayers(2)

    player1 := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Alice"))
    player2 := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Bob"))
    player3 := entity.NewPlayer(valueobject.NewPlayerID(), valueobject.NewPlayerName("Charlie"))

    room.AddPlayer(player1)
    room.AddPlayer(player2)

    // Act
    err := room.AddPlayer(player3)

    // Assert
    assert.Error(t, err)
    assert.Equal(t, domain.ErrRoomFull, err)
}
```

### 2. テーブル駆動テスト

```go
func TestScoreCalculation(t *testing.T) {
    tests := []struct {
        name     string
        category valueobject.ScoreCategory
        diceSet  valueobject.DiceSet
        expected valueobject.Score
    }{
        {
            name:     "Ones - two ones",
            category: valueobject.Ones,
            diceSet:  valueobject.NewDiceSetFromValues([]int{1, 1, 2, 3, 4}),
            expected: valueobject.NewScore(2),
        },
        {
            name:     "ThreeOfAKind - valid",
            category: valueobject.ThreeOfAKind,
            diceSet:  valueobject.NewDiceSetFromValues([]int{3, 3, 3, 1, 2}),
            expected: valueobject.NewScore(12),
        },
        {
            name:     "ThreeOfAKind - invalid",
            category: valueobject.ThreeOfAKind,
            diceSet:  valueobject.NewDiceSetFromValues([]int{1, 2, 3, 4, 5}),
            expected: valueobject.NewScore(0),
        },
        {
            name:     "FullHouse - valid",
            category: valueobject.FullHouse,
            diceSet:  valueobject.NewDiceSetFromValues([]int{2, 2, 3, 3, 3}),
            expected: valueobject.NewScore(25),
        },
        {
            name:     "Yahtzee",
            category: valueobject.Yahtzee,
            diceSet:  valueobject.NewDiceSetFromValues([]int{5, 5, 5, 5, 5}),
            expected: valueobject.NewScore(50),
        },
    }

    scoreCalc := service.NewScoreCalculationService()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            score := scoreCalc.CalculateScore(tt.category, tt.diceSet)

            // Assert
            assert.Equal(t, tt.expected, score)
        })
    }
}
```

### 3. アプリケーション層のテスト

```go
// internal/application/usecase/room/create_room_test.go
package room_test

import (
    "context"
    "testing"

    "github.com/edge2992/yatzcli/internal/application/usecase/room"
    "github.com/edge2992/yatzcli/internal/domain/valueobject"
    "github.com/edge2992/yatzcli/test/mock"
    "github.com/stretchr/testify/assert"
)

func TestCreateRoomUseCase_Execute_Success(t *testing.T) {
    // Arrange
    mockRepo := mock.NewMockRoomRepository()
    mockValidator := mock.NewMockGameRuleValidator()
    mockEventBus := mock.NewMockEventBus()

    usecase := room.NewCreateRoomUseCase(mockRepo, mockValidator, mockEventBus)

    input := room.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName("Alice"),
        MaxPlayers:     4,
    }

    // Act
    output, err := usecase.Execute(context.Background(), input)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, output.RoomID)
    assert.NotEmpty(t, output.PlayerID)

    // モック検証
    assert.True(t, mockRepo.SaveCalled)
    assert.Equal(t, 1, mockEventBus.PublishedEventCount())

    // イベント検証
    events := mockEventBus.PublishedEvents()
    assert.Equal(t, "RoomCreatedEvent", events[0].EventType())
}

func TestCreateRoomUseCase_Execute_RepositoryError(t *testing.T) {
    // Arrange
    mockRepo := mock.NewMockRoomRepository()
    mockRepo.SaveReturnError = errors.New("database error")

    usecase := room.NewCreateRoomUseCase(mockRepo, nil, nil)

    input := room.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName("Alice"),
        MaxPlayers:     4,
    }

    // Act
    _, err := usecase.Execute(context.Background(), input)

    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "database error")
}
```

### 4. 統合テスト

```go
// test/integration/room_integration_test.go
//go:build integration

package integration_test

import (
    "context"
    "testing"

    "github.com/edge2992/yatzcli/internal/application/usecase/room"
    "github.com/edge2992/yatzcli/internal/domain/service"
    "github.com/edge2992/yatzcli/internal/infrastructure/persistence/memory"
    "github.com/edge2992/yatzcli/test/fake"
    "github.com/stretchr/testify/assert"
)

func TestRoomCreationAndJoining_Integration(t *testing.T) {
    // Arrange: 実際の依存関係を使用
    roomRepo := memory.NewRoomRepository()
    playerRepo := memory.NewPlayerRepository()
    eventBus := fake.NewFakeEventBus()
    validator := service.NewGameRuleValidator()

    createRoomUC := room.NewCreateRoomUseCase(roomRepo, validator, eventBus)
    joinRoomUC := room.NewJoinRoomUseCase(roomRepo, playerRepo, validator, eventBus)

    // Act: ルーム作成
    createOutput, err := createRoomUC.Execute(context.Background(), room.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName("Alice"),
        MaxPlayers:     4,
    })
    assert.NoError(t, err)

    // Act: 別のプレイヤーが参加
    joinOutput, err := joinRoomUC.Execute(context.Background(), room.JoinRoomInput{
        RoomID:     createOutput.RoomID,
        PlayerName: valueobject.NewPlayerName("Bob"),
    })
    assert.NoError(t, err)

    // Assert: ルームの状態確認
    savedRoom, err := roomRepo.FindByID(createOutput.RoomID)
    assert.NoError(t, err)
    assert.Equal(t, 2, len(savedRoom.GetPlayers()))

    // Assert: イベント確認
    assert.Equal(t, 2, eventBus.EventCount()) // RoomCreated + PlayerJoined
}
```

### 5. E2Eテスト

```go
// test/e2e/full_game_test.go
//go:build e2e

package e2e_test

import (
    "testing"
    "time"

    "github.com/edge2992/yatzcli/internal/infrastructure/client"
    "github.com/edge2992/yatzcli/internal/infrastructure/server"
    "github.com/stretchr/testify/assert"
)

func TestFullGameFlow_E2E(t *testing.T) {
    // Arrange: サーバー起動
    srv := server.NewTCPServer(":18080", setupHandler())
    go srv.Start()
    defer srv.Stop()

    time.Sleep(100 * time.Millisecond) // サーバー起動待機

    // クライアント接続
    client1 := client.NewTCPClient("localhost:18080")
    client2 := client.NewTCPClient("localhost:18080")

    err := client1.Connect()
    assert.NoError(t, err)
    defer client1.Disconnect()

    err = client2.Connect()
    assert.NoError(t, err)
    defer client2.Disconnect()

    // Act: ルーム作成
    createResp, err := client1.CreateRoom("Alice", 2)
    assert.NoError(t, err)
    roomID := createResp.RoomID

    // Act: 参加
    _, err = client2.JoinRoom(roomID, "Bob")
    assert.NoError(t, err)

    // Act: ゲーム開始
    _, err = client1.StartGame(roomID)
    assert.NoError(t, err)

    // Act: 1ターン実行
    _, err = client1.RollDice(roomID, []int{})
    assert.NoError(t, err)

    _, err = client1.ChooseCategory(roomID, "Ones")
    assert.NoError(t, err)

    // Assert: ターン交代確認
    // （省略: 実際には複数ターンを実行し、ゲーム終了まで確認）
}
```

---

## テストヘルパーとフィクスチャ

### 1. テストビルダー

```go
// test/builder/room_builder.go
package builder

type RoomBuilder struct {
    room *entity.Room
}

func NewRoomBuilder() *RoomBuilder {
    return &RoomBuilder{
        room: entity.NewRoom(valueobject.NewRoomID()),
    }
}

func (b *RoomBuilder) WithMaxPlayers(max int) *RoomBuilder {
    b.room.SetMaxPlayers(max)
    return b
}

func (b *RoomBuilder) WithPlayer(name string) *RoomBuilder {
    player := entity.NewPlayer(
        valueobject.NewPlayerID(),
        valueobject.NewPlayerName(name),
    )
    b.room.AddPlayer(player)
    return b
}

func (b *RoomBuilder) WithGameStarted() *RoomBuilder {
    b.room.StartGame()
    return b
}

func (b *RoomBuilder) Build() *entity.Room {
    return b.room
}

// 使用例
room := NewRoomBuilder().
    WithMaxPlayers(4).
    WithPlayer("Alice").
    WithPlayer("Bob").
    WithGameStarted().
    Build()
```

### 2. テストフィクスチャ

```go
// test/fixtures/fixtures.go
package fixtures

func CreateTestRoom() *entity.Room {
    room := entity.NewRoom(valueobject.RoomID("test-room-1"))
    room.SetMaxPlayers(4)
    return room
}

func CreateTestPlayer(name string) *entity.Player {
    return entity.NewPlayer(
        valueobject.PlayerID(fmt.Sprintf("player-%s", name)),
        valueobject.NewPlayerName(name),
    )
}

func CreateRoomWithPlayers(playerNames ...string) *entity.Room {
    room := CreateTestRoom()
    for _, name := range playerNames {
        room.AddPlayer(CreateTestPlayer(name))
    }
    return room
}

func CreateInProgressRoom() *entity.Room {
    room := CreateRoomWithPlayers("Alice", "Bob")
    room.StartGame()
    return room
}
```

---

## 並行性テスト

### 1. データ競合の検出

```go
func TestRoom_ConcurrentAccess(t *testing.T) {
    // go test -race を使用
    room := entity.NewRoom(valueobject.NewRoomID())

    var wg sync.WaitGroup
    errors := make(chan error, 10)

    // 複数のゴルーチンから同時にプレイヤーを追加
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()

            player := entity.NewPlayer(
                valueobject.NewPlayerID(),
                valueobject.NewPlayerName(fmt.Sprintf("Player%d", index)),
            )

            if err := room.AddPlayer(player); err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    // エラー確認
    errorCount := 0
    for range errors {
        errorCount++
    }

    // 最大プレイヤー数を超えた分はエラーになるべき
    assert.True(t, errorCount > 0)
}
```

### 2. タイムアウトテスト

```go
func TestUseCase_ExecutionTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    // 遅いリポジトリをシミュレート
    slowRepo := &mock.MockRoomRepository{
        SaveDelay: 200 * time.Millisecond,
    }

    usecase := room.NewCreateRoomUseCase(slowRepo, nil, nil)

    _, err := usecase.Execute(ctx, room.CreateRoomInput{
        HostPlayerName: valueobject.NewPlayerName("Alice"),
    })

    assert.Error(t, err)
    assert.Equal(t, context.DeadlineExceeded, err)
}
```

---

## テストカバレッジ目標

| レイヤー | 目標カバレッジ | 優先度 |
|---------|---------------|--------|
| ドメイン層 | 90%以上 | 最高 |
| アプリケーション層 | 80%以上 | 高 |
| インフラ層 | 70%以上 | 中 |
| プレゼンテーション層 | 60%以上 | 中 |

### カバレッジ測定

```bash
# カバレッジ測定
go test ./internal/... -coverprofile=coverage.out

# HTML形式で表示
go tool cover -html=coverage.out

# パッケージ別のカバレッジ
go test ./internal/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## CI/CDでのテスト実行

### GitHub Actions例

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run unit tests
        run: go test ./internal/... -race -coverprofile=coverage.out

      - name: Run integration tests
        run: go test ./test/integration/... -tags=integration

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

      - name: Check coverage threshold
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage $coverage% is below 80%"
            exit 1
          fi
```

---

## まとめ

このテスタビリティ設計により：

✅ **高いテストカバレッジ**: 各レイヤーを網羅的にテスト
✅ **独立したテスト**: モック・スタブによる分離
✅ **保守性**: テストヘルパーによる再利用性
✅ **並行性の検証**: データ競合の早期発見
✅ **CI/CD統合**: 自動化されたテスト実行
✅ **品質保証**: 高いコード品質の維持

すべての設計ドキュメントが完成しました！
