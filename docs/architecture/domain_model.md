# ドメインモデル設計書

## 概要

本ドキュメントはYatzCLIのドメイン駆動設計（DDD）に基づくドメインモデルを定義します。Yahtzeeゲームのビジネスロジックとルールをドメイン層に集約し、技術的関心事から分離します。

## ドメイン層の目的

- ゲームのビジネスルールと不変条件を保護
- 技術的詳細（ネットワーク、永続化、UI）から独立
- ユビキタス言語でドメイン知識を表現
- 高いテスタビリティとメンテナンス性

---

## エンティティ（Entity）

エンティティは識別子を持ち、ライフサイクルを通じて同一性を保持するオブジェクトです。

### 1. Room（集約ルート）

**責務**: ゲームセッション全体の状態管理と整合性の保証

```go
package domain

type Room struct {
    // 識別子
    id RoomID

    // 集約内のエンティティ
    players map[PlayerID]*Player
    host    PlayerID

    // 値オブジェクト
    state   GameState
    turn    *Turn

    // メタ情報
    createdAt time.Time
    updatedAt time.Time

    // 並行性制御
    mu sync.RWMutex
}
```

**不変条件（Invariants）**:
- ルームIDは一意でnilでない
- プレイヤー数は1〜MaxPlayers（デフォルト4）
- ゲーム開始後はプレイヤーの追加・削除不可
- 現在のターンプレイヤーは必ずルーム内に存在

**ドメインメソッド**:
```go
// プレイヤー管理
func (r *Room) AddPlayer(player *Player) error
func (r *Room) RemovePlayer(playerID PlayerID) error
func (r *Room) GetPlayer(playerID PlayerID) (*Player, error)

// ゲームライフサイクル
func (r *Room) StartGame() error
func (r *Room) EndGame() error
func (r *Room) IsGameStarted() bool
func (r *Room) CanStartGame() bool

// ターン管理
func (r *Room) StartNextTurn() error
func (r *Room) GetCurrentPlayer() (*Player, error)

// 状態取得
func (r *Room) GetState() GameState
func (r *Room) GetPlayers() []*Player
```

### 2. Player

**責務**: プレイヤーの情報とスコアカードの管理

```go
type Player struct {
    // 識別子
    id   PlayerID
    name PlayerName

    // スコア情報
    scoreCard *ScoreCard

    // 接続情報（インフラ層への参照は避ける）
    connectionID ConnectionID

    // メタ情報
    joinedAt time.Time
    isActive bool
}
```

**不変条件**:
- プレイヤーIDとプレイヤー名はnilでない
- スコアカードは必ず存在
- 名前は1〜20文字

**ドメインメソッド**:
```go
func (p *Player) RecordScore(category ScoreCategory, score Score) error
func (p *Player) HasScoredCategory(category ScoreCategory) bool
func (p *Player) CalculateTotalScore() Score
func (p *Player) GetScoreCard() *ScoreCard
func (p *Player) GetAvailableCategories() []ScoreCategory
```

### 3. ScoreCard

**責務**: スコアシートの管理とボーナス計算

```go
type ScoreCard struct {
    scores      map[ScoreCategory]Score
    upperBonus  Score
    totalScore  Score

    // キャッシュの無効化フラグ
    dirty bool
}
```

**不変条件**:
- 各カテゴリーは一度のみスコア記録可能
- 上段ボーナスは63点以上で自動付与

**ドメインメソッド**:
```go
func (sc *ScoreCard) RecordScore(category ScoreCategory, score Score) error
func (sc *ScoreCard) GetScore(category ScoreCategory) (Score, bool)
func (sc *ScoreCard) CalculateUpperBonus() Score
func (sc *ScoreCard) CalculateTotalScore() Score
func (sc *ScoreCard) IsComplete() bool
```

---

## 値オブジェクト（Value Object）

値オブジェクトは不変で、属性の値によって等価性が判断されます。

### 1. RoomID

```go
type RoomID string

func NewRoomID() RoomID
func ParseRoomID(s string) (RoomID, error)
func (id RoomID) String() string
func (id RoomID) IsValid() bool
```

**制約**: 空文字列でない、UUID形式

### 2. PlayerID

```go
type PlayerID string

func NewPlayerID() PlayerID
func (id PlayerID) String() string
```

### 3. PlayerName

```go
type PlayerName struct {
    value string
}

func NewPlayerName(name string) (PlayerName, error)
func (n PlayerName) String() string
func (n PlayerName) Equals(other PlayerName) bool
```

**制約**: 1〜20文字、空白のみは不可

### 4. Dice

```go
type Dice struct {
    value int    // 1-6
    held  bool   // ホールド状態
}

func NewDice(value int) (Dice, error)
func (d Dice) Value() int
func (d Dice) IsHeld() bool
func (d Dice) Hold() Dice
func (d Dice) Release() Dice
func (d Dice) Roll() Dice
```

**制約**: 値は1〜6、不変性を保証

### 5. DiceSet

```go
type DiceSet struct {
    dices [5]Dice
}

func NewDiceSet() DiceSet
func (ds DiceSet) Roll() DiceSet
func (ds DiceSet) RollUnheld() DiceSet
func (ds DiceSet) GetDice(index int) (Dice, error)
func (ds DiceSet) HoldDice(indices []int) (DiceSet, error)
func (ds DiceSet) Values() []int
```

### 6. Score

```go
type Score struct {
    value int
}

func NewScore(value int) Score
func (s Score) Value() int
func (s Score) Add(other Score) Score
func (s Score) IsZero() bool
```

### 7. ScoreCategory

```go
type ScoreCategory int

const (
    // 上段（Upper Section）
    Ones ScoreCategory = iota
    Twos
    Threes
    Fours
    Fives
    Sixes

    // 下段（Lower Section）
    ThreeOfAKind
    FourOfAKind
    FullHouse
    SmallStraight
    LargeStraight
    Yahtzee
    Chance
)

func (sc ScoreCategory) IsUpperSection() bool
func (sc ScoreCategory) String() string
```

### 8. Turn

**責務**: 1ターンの状態管理

```go
type Turn struct {
    playerID    PlayerID
    diceSet     DiceSet
    rollCount   int
    phase       TurnPhase
}

const (
    RollPhase TurnPhase = iota
    CategorySelectionPhase
)

func NewTurn(playerID PlayerID) *Turn
func (t *Turn) CanRoll() bool
func (t *Turn) Roll() error
func (t *Turn) RollCount() int
func (t *Turn) GetDiceSet() DiceSet
func (t *Turn) SelectCategory(category ScoreCategory) error
```

**不変条件**:
- ロール回数は0〜3
- フェーズの遷移は RollPhase → CategorySelectionPhase のみ

### 9. GameState

```go
type GameState int

const (
    WaitingForPlayers GameState = iota
    InProgress
    Finished
)

func (gs GameState) String() string
func (gs GameState) CanTransitionTo(next GameState) bool
```

**状態遷移**:
```
WaitingForPlayers → InProgress → Finished
```

---

## 集約（Aggregate）

### Room集約

**集約ルート**: Room
**集約内エンティティ**: Player, ScoreCard
**値オブジェクト**: Turn, DiceSet, GameState

**境界**:
- Roomを通じてのみPlayersにアクセス
- Roomが集約内の一貫性を保証
- トランザクション境界はRoom単位

**不変条件の保護**:
```go
func (r *Room) AddPlayer(player *Player) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // 不変条件チェック
    if r.state != WaitingForPlayers {
        return ErrGameAlreadyStarted
    }
    if len(r.players) >= MaxPlayers {
        return ErrRoomFull
    }
    if _, exists := r.players[player.id]; exists {
        return ErrPlayerAlreadyInRoom
    }

    r.players[player.id] = player
    r.updatedAt = time.Now()
    return nil
}
```

---

## ドメインサービス（Domain Service）

エンティティや値オブジェクトに所属しない振る舞いを定義します。

### 1. ScoreCalculationService

**責務**: スコア計算ロジックの実装

```go
type ScoreCalculationService interface {
    CalculateScore(category ScoreCategory, diceSet DiceSet) Score
    ValidateCategoryScore(category ScoreCategory, diceSet DiceSet) error
}

type scoreCalculationService struct {
    calculators map[ScoreCategory]ScoreCalculator
}

type ScoreCalculator func(DiceSet) Score
```

**計算ロジック**:
- 各カテゴリーのルールに基づいた計算
- Strategy パターンで拡張可能

### 2. GameRuleValidator

**責務**: ゲームルールの検証

```go
type GameRuleValidator interface {
    CanStartGame(room *Room) error
    CanRollDice(turn *Turn) error
    CanSelectCategory(player *Player, category ScoreCategory) error
}
```

### 3. TurnManager

**責務**: ターン順序の管理

```go
type TurnManager interface {
    GetNextPlayer(room *Room, currentPlayer PlayerID) (PlayerID, error)
    IsGameComplete(room *Room) bool
    CalculateFinalRankings(room *Room) []PlayerRanking
}
```

---

## リポジトリインターフェース（Repository）

ドメイン層はインターフェースのみ定義、実装はインフラ層が提供します。

### 1. RoomRepository

```go
package domain

type RoomRepository interface {
    // CRUD操作
    Save(room *Room) error
    FindByID(id RoomID) (*Room, error)
    FindAll() ([]*Room, error)
    Delete(id RoomID) error

    // クエリ
    FindByPlayerID(playerID PlayerID) ([]*Room, error)
    FindAvailableRooms() ([]*Room, error)

    // トランザクション
    BeginTransaction() (Transaction, error)
}
```

### 2. PlayerRepository

```go
type PlayerRepository interface {
    Save(player *Player) error
    FindByID(id PlayerID) (*Player, error)
    FindByName(name PlayerName) (*Player, error)
    Delete(id PlayerID) error
}
```

---

## ドメインイベント（Domain Event）

状態変化を表現するイベントを定義します（オプション）。

```go
type DomainEvent interface {
    OccurredAt() time.Time
    EventType() string
}

// 具体的なイベント
type GameStartedEvent struct {
    roomID    RoomID
    playerIDs []PlayerID
    timestamp time.Time
}

type TurnStartedEvent struct {
    roomID   RoomID
    playerID PlayerID
    turnNum  int
    timestamp time.Time
}

type ScoreRecordedEvent struct {
    roomID   RoomID
    playerID PlayerID
    category ScoreCategory
    score    Score
    timestamp time.Time
}

type GameFinishedEvent struct {
    roomID   RoomID
    rankings []PlayerRanking
    timestamp time.Time
}
```

---

## ドメインエラー

ドメイン層のエラーを型として定義します。

```go
package domain

import "errors"

var (
    // Room関連
    ErrRoomNotFound         = errors.New("room not found")
    ErrRoomFull             = errors.New("room is full")
    ErrGameAlreadyStarted   = errors.New("game already started")
    ErrGameNotStarted       = errors.New("game not started")

    // Player関連
    ErrPlayerNotFound       = errors.New("player not found")
    ErrPlayerAlreadyInRoom  = errors.New("player already in room")
    ErrNotPlayerTurn        = errors.New("not player's turn")

    // Turn関連
    ErrMaxRollsExceeded     = errors.New("maximum rolls exceeded")
    ErrInvalidTurnPhase     = errors.New("invalid turn phase")

    // Score関連
    ErrCategoryAlreadyScored = errors.New("category already scored")
    ErrInvalidCategory      = errors.New("invalid score category")
    ErrInvalidScore         = errors.New("invalid score value")

    // Dice関連
    ErrInvalidDiceValue     = errors.New("dice value must be 1-6")
    ErrInvalidDiceIndex     = errors.New("invalid dice index")

    // Validation関連
    ErrInvalidPlayerName    = errors.New("player name must be 1-20 characters")
    ErrInvalidRoomID        = errors.New("invalid room ID format")
)

// カスタムエラー型（より詳細な情報が必要な場合）
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}
```

---

## ドメインモデルの使用例

### ゲーム開始フロー

```go
// 1. ルーム作成
room := domain.NewRoom(domain.NewRoomID())

// 2. プレイヤー追加
player1 := domain.NewPlayer(domain.NewPlayerID(), domain.NewPlayerName("Alice"))
player2 := domain.NewPlayer(domain.NewPlayerID(), domain.NewPlayerName("Bob"))

room.AddPlayer(player1)
room.AddPlayer(player2)

// 3. ゲーム開始
if err := room.StartGame(); err != nil {
    return err
}

// 4. ターン開始
turn := room.GetCurrentTurn()
currentPlayer := room.GetCurrentPlayer()
```

### スコア記録フロー

```go
// 1. ダイスロール
turn.Roll()
diceSet := turn.GetDiceSet()

// 2. スコア計算
scoreService := NewScoreCalculationService()
score := scoreService.CalculateScore(domain.ThreeOfAKind, diceSet)

// 3. カテゴリー選択とスコア記録
player.RecordScore(domain.ThreeOfAKind, score)

// 4. 次のターンへ
room.StartNextTurn()
```

---

## まとめ

このドメインモデル設計により：

✅ **ビジネスロジックの集約**: ゲームルールがドメイン層に集中
✅ **不変条件の保護**: エンティティと集約が一貫性を保証
✅ **技術的独立性**: インフラやUIから独立したドメイン層
✅ **テスタビリティ**: ドメインロジックを単独でテスト可能
✅ **拡張性**: 新しいルールやカテゴリーの追加が容易
✅ **並行性安全**: 集約ルートレベルでの排他制御

次のステップ: レイヤーアーキテクチャ設計書でドメイン層と他の層の関係を定義します。
