# クラス図・シーケンス図

## 概要

本ドキュメントはYatzCLIの視覚的な設計図を提供します。Mermaid形式で記述されたクラス図、シーケンス図、状態遷移図を含みます。

---

## 1. ドメイン層クラス図

### 1.1 集約とエンティティ

```mermaid
classDiagram
    class Room {
        <<Aggregate Root>>
        -RoomID id
        -map~PlayerID, Player~ players
        -PlayerID host
        -RoomState state
        -Turn turn
        -int maxPlayers
        -time createdAt
        -RWMutex mu
        +AddPlayer(player) error
        +RemovePlayer(playerID) error
        +StartGame() error
        +EndGame() error
        +GetCurrentPlayer() Player
        +GetState() RoomState
        +IsGameComplete() bool
    }

    class Player {
        <<Entity>>
        -PlayerID id
        -PlayerName name
        -ScoreCard scoreCard
        -ConnectionID connectionID
        -PlayerState state
        -time lastSeenAt
        +RecordScore(category, score) error
        +CalculateTotalScore() Score
        +GetAvailableCategories() []ScoreCategory
        +Disconnect()
        +Reconnect(connID) error
    }

    class ScoreCard {
        <<Entity>>
        -map~ScoreCategory, Score~ scores
        -Score upperBonus
        -Score totalScore
        -bool dirty
        +RecordScore(category, score) error
        +GetScore(category) Score
        +CalculateUpperBonus() Score
        +CalculateTotalScore() Score
        +IsComplete() bool
    }

    class Turn {
        <<Value Object>>
        -PlayerID playerID
        -DiceSet diceSet
        -int rollCount
        -TurnPhase phase
        -time startedAt
        +Roll() error
        +HoldDice(indices) error
        +CanRoll() bool
        +SelectCategory(category) error
    }

    Room "1" *-- "1..*" Player : contains
    Player "1" *-- "1" ScoreCard : has
    Room "1" o-- "0..1" Turn : current turn
```

### 1.2 値オブジェクト

```mermaid
classDiagram
    class RoomID {
        <<Value Object>>
        -string value
        +NewRoomID() RoomID
        +String() string
        +IsValid() bool
    }

    class PlayerID {
        <<Value Object>>
        -string value
        +NewPlayerID() PlayerID
        +String() string
    }

    class PlayerName {
        <<Value Object>>
        -string value
        +NewPlayerName(name) PlayerName
        +String() string
        +Equals(other) bool
    }

    class Dice {
        <<Value Object>>
        -int value
        -bool held
        +NewDice(value) Dice
        +Value() int
        +IsHeld() bool
        +Hold() Dice
        +Release() Dice
        +Roll() Dice
    }

    class DiceSet {
        <<Value Object>>
        -[5]Dice dices
        +NewDiceSet() DiceSet
        +Roll() DiceSet
        +RollUnheld() DiceSet
        +HoldDice(indices) DiceSet
        +Values() []int
    }

    class Score {
        <<Value Object>>
        -int value
        +NewScore(value) Score
        +Value() int
        +Add(other) Score
    }

    DiceSet "1" *-- "5" Dice : contains
```

### 1.3 ドメインサービス

```mermaid
classDiagram
    class ScoreCalculationService {
        <<Service>>
        -map~ScoreCategory, ScoreCalculator~ calculators
        +CalculateScore(category, diceSet) Score
        +ValidateCategoryScore(category, diceSet) error
    }

    class GameRuleValidator {
        <<Service>>
        +CanStartGame(room) error
        +CanRollDice(turn) error
        +CanSelectCategory(player, category) error
        +ValidatePlayerCount(count) error
    }

    class TurnManager {
        <<Service>>
        +GetNextPlayer(room, currentPlayer) PlayerID
        +IsGameComplete(room) bool
        +CalculateFinalRankings(room) []PlayerRanking
    }
```

### 1.4 リポジトリインターフェース

```mermaid
classDiagram
    class RoomRepository {
        <<Interface>>
        +Save(room) error
        +FindByID(id) Room
        +FindAll() []Room
        +Delete(id) error
        +FindByPlayerID(playerID) []Room
    }

    class PlayerRepository {
        <<Interface>>
        +Save(player) error
        +FindByID(id) Player
        +FindByName(name) Player
        +Delete(id) error
    }

    Room ..> RoomRepository : persisted by
    Player ..> PlayerRepository : persisted by
```

---

## 2. アプリケーション層クラス図

```mermaid
classDiagram
    class CreateRoomUseCase {
        -RoomRepository roomRepo
        -GameRuleValidator validator
        -EventBus eventBus
        +Execute(input) output, error
    }

    class JoinRoomUseCase {
        -RoomRepository roomRepo
        -PlayerRepository playerRepo
        -GameRuleValidator validator
        -EventBus eventBus
        +Execute(input) output, error
    }

    class RollDiceUseCase {
        -RoomRepository roomRepo
        -TurnManager turnMgr
        -GameRuleValidator validator
        -EventBus eventBus
        +Execute(input) output, error
    }

    class ChooseCategoryUseCase {
        -RoomRepository roomRepo
        -ScoreCalculationService scoreCalc
        -GameRuleValidator validator
        -EventBus eventBus
        +Execute(input) output, error
    }

    CreateRoomUseCase ..> RoomRepository : uses
    CreateRoomUseCase ..> GameRuleValidator : uses
    JoinRoomUseCase ..> RoomRepository : uses
    JoinRoomUseCase ..> PlayerRepository : uses
    RollDiceUseCase ..> RoomRepository : uses
    RollDiceUseCase ..> TurnManager : uses
    ChooseCategoryUseCase ..> RoomRepository : uses
    ChooseCategoryUseCase ..> ScoreCalculationService : uses
```

---

## 3. インフラ層クラス図

```mermaid
classDiagram
    class TCPServer {
        -string address
        -Listener listener
        -ConnectionManager connMgr
        -MessageHandler handler
        +Start() error
        +Stop() error
        -handleConnection(conn)
    }

    class ConnectionManager {
        -map~ConnectionID, Connection~ connections
        -RWMutex mu
        +Register(conn) ConnectionID
        +Unregister(connID)
        +GetConnection(connID) Connection
        +BroadcastToRoom(roomID, message)
    }

    class Connection {
        <<Interface>>
        +Send(data) error
        +Receive() []byte, error
        +Close() error
        +RemoteAddr() string
    }

    class TCPConnection {
        -net.Conn conn
        -MessageCodec codec
        +Send(data) error
        +Receive() []byte, error
        +Close() error
    }

    class MessageCodec {
        <<Interface>>
        +Encode(message) []byte, error
        +Decode(data) Message, error
    }

    class JSONCodec {
        +Encode(message) []byte, error
        +Decode(data) Message, error
    }

    TCPServer "1" --> "1" ConnectionManager : manages
    ConnectionManager "1" --> "*" Connection : holds
    Connection <|.. TCPConnection : implements
    TCPConnection "1" --> "1" MessageCodec : uses
    MessageCodec <|.. JSONCodec : implements
```

---

## 4. レイヤー間の関係図

```mermaid
classDiagram
    class MessageHandler {
        <<Presentation>>
        +HandleMessage(conn, msg) error
    }

    class CreateRoomHandler {
        <<Presentation>>
        -CreateRoomUseCase usecase
        +Handle(conn, msg) error
    }

    class CreateRoomUseCase {
        <<Application>>
        -RoomRepository roomRepo
        +Execute(input) output, error
    }

    class RoomRepository {
        <<Domain Interface>>
    }

    class InMemoryRoomRepository {
        <<Infrastructure>>
        -map rooms
        -RWMutex mu
        +Save(room) error
        +FindByID(id) Room
    }

    class Room {
        <<Domain Entity>>
    }

    MessageHandler <|-- CreateRoomHandler : implements
    CreateRoomHandler --> CreateRoomUseCase : uses
    CreateRoomUseCase --> RoomRepository : depends on
    RoomRepository <|.. InMemoryRoomRepository : implements
    InMemoryRoomRepository --> Room : stores
```

---

## 5. シーケンス図

### 5.1 ルーム作成フロー

```mermaid
sequenceDiagram
    actor Client
    participant Handler as CreateRoomHandler
    participant UseCase as CreateRoomUseCase
    participant Room as Room
    participant Repo as RoomRepository
    participant EventBus as EventBus

    Client->>Handler: CreateRoomCommand
    activate Handler

    Handler->>UseCase: Execute(input)
    activate UseCase

    UseCase->>Room: NewRoom(roomID)
    activate Room
    Room-->>UseCase: room
    deactivate Room

    UseCase->>Room: AddPlayer(host)
    activate Room
    Room->>Room: validate constraints
    Room-->>UseCase: nil (success)
    deactivate Room

    UseCase->>Repo: Save(room)
    activate Repo
    Repo-->>UseCase: nil
    deactivate Repo

    UseCase->>EventBus: Publish(RoomCreatedEvent)
    activate EventBus
    EventBus-->>UseCase:
    deactivate EventBus

    UseCase-->>Handler: output
    deactivate UseCase

    Handler->>Client: SuccessResponse
    deactivate Handler
```

### 5.2 ゲーム開始フロー

```mermaid
sequenceDiagram
    actor Host
    participant Handler as StartGameHandler
    participant UseCase as StartGameUseCase
    participant Repo as RoomRepository
    participant Room as Room
    participant EventBus as EventBus
    actor Players

    Host->>Handler: StartGameCommand
    activate Handler

    Handler->>UseCase: Execute(roomID)
    activate UseCase

    UseCase->>Repo: FindByID(roomID)
    activate Repo
    Repo-->>UseCase: room
    deactivate Repo

    UseCase->>Room: StartGame()
    activate Room
    Room->>Room: validate (>= 2 players)
    Room->>Room: state = InProgress
    Room->>Room: create first turn
    Room-->>UseCase: nil (success)
    deactivate Room

    UseCase->>Repo: Save(room)
    activate Repo
    Repo-->>UseCase: nil
    deactivate Repo

    UseCase->>EventBus: Publish(GameStartedEvent)
    activate EventBus
    EventBus-->>UseCase:
    deactivate EventBus

    UseCase-->>Handler: output
    deactivate UseCase

    Handler->>Host: SuccessResponse
    Handler->>Players: GameStartedEvent (broadcast)
    Handler->>Players: TurnStartedEvent (broadcast)
    deactivate Handler
```

### 5.3 ダイスロールフロー

```mermaid
sequenceDiagram
    actor Player
    participant Handler as RollDiceHandler
    participant UseCase as RollDiceUseCase
    participant Repo as RoomRepository
    participant Room as Room
    participant Turn as Turn
    participant EventBus as EventBus
    actor AllPlayers

    Player->>Handler: RollDiceCommand<br/>{heldIndices: [0,2]}
    activate Handler

    Handler->>UseCase: Execute(roomID, playerID, heldIndices)
    activate UseCase

    UseCase->>Repo: FindByID(roomID)
    activate Repo
    Repo-->>UseCase: room
    deactivate Repo

    UseCase->>Room: GetCurrentTurn()
    activate Room
    Room-->>UseCase: turn
    deactivate Room

    UseCase->>Turn: HoldDice(heldIndices)
    activate Turn
    Turn-->>UseCase: nil
    deactivate Turn

    UseCase->>Turn: Roll()
    activate Turn
    Turn->>Turn: validate (rollCount < 3)
    Turn->>Turn: roll unheld dice
    Turn->>Turn: rollCount++
    Turn-->>UseCase: nil (success)
    deactivate Turn

    UseCase->>Repo: Save(room)
    activate Repo
    Repo-->>UseCase: nil
    deactivate Repo

    UseCase->>EventBus: Publish(DiceRolledEvent)
    activate EventBus
    EventBus-->>UseCase:
    deactivate EventBus

    UseCase-->>Handler: output {diceSet, rollCount, canRoll}
    deactivate UseCase

    Handler->>AllPlayers: DiceRolledEvent (broadcast)
    Handler->>Player: SuccessResponse
    deactivate Handler
```

### 5.4 カテゴリー選択・スコア記録フロー

```mermaid
sequenceDiagram
    actor Player
    participant Handler as ChooseCategoryHandler
    participant UseCase as ChooseCategoryUseCase
    participant ScoreCalc as ScoreCalculationService
    participant Repo as RoomRepository
    participant Room as Room
    participant Player as PlayerEntity
    participant TurnMgr as TurnManager
    participant EventBus as EventBus
    actor AllPlayers

    Player->>Handler: ChooseCategoryCommand<br/>{category: "ThreeOfAKind"}
    activate Handler

    Handler->>UseCase: Execute(roomID, playerID, category)
    activate UseCase

    UseCase->>Repo: FindByID(roomID)
    Repo-->>UseCase: room

    UseCase->>Room: GetCurrentPlayer()
    Room-->>UseCase: player

    UseCase->>Room: GetCurrentTurn()
    Room-->>UseCase: turn

    UseCase->>ScoreCalc: CalculateScore(category, diceSet)
    activate ScoreCalc
    ScoreCalc-->>UseCase: score
    deactivate ScoreCalc

    UseCase->>PlayerEntity: RecordScore(category, score)
    activate PlayerEntity
    PlayerEntity->>PlayerEntity: validate category not used
    PlayerEntity->>PlayerEntity: update scoreCard
    PlayerEntity-->>UseCase: nil (success)
    deactivate PlayerEntity

    UseCase->>TurnMgr: GetNextPlayer(room, currentPlayer)
    activate TurnMgr
    TurnMgr-->>UseCase: nextPlayerID
    deactivate TurnMgr

    UseCase->>Room: StartNextTurn(nextPlayerID)
    activate Room
    Room->>Room: create new Turn
    Room-->>UseCase: nil
    deactivate Room

    UseCase->>Repo: Save(room)
    Repo-->>UseCase: nil

    UseCase->>EventBus: Publish(CategoryChosenEvent)
    UseCase->>EventBus: Publish(ScoreboardUpdatedEvent)
    UseCase->>EventBus: Publish(TurnStartedEvent)

    UseCase-->>Handler: output
    deactivate UseCase

    Handler->>AllPlayers: CategoryChosenEvent (broadcast)
    Handler->>AllPlayers: ScoreboardUpdatedEvent (broadcast)
    Handler->>AllPlayers: TurnStartedEvent (broadcast)
    Handler->>Player: SuccessResponse
    deactivate Handler
```

### 5.5 ゲーム終了フロー

```mermaid
sequenceDiagram
    actor Player
    participant UseCase as ChooseCategoryUseCase
    participant Room as Room
    participant TurnMgr as TurnManager
    participant EventBus as EventBus
    actor AllPlayers

    Player->>UseCase: Execute (last category)
    activate UseCase

    UseCase->>Room: RecordLastScore()
    UseCase->>TurnMgr: IsGameComplete(room)
    activate TurnMgr
    TurnMgr->>Room: check all scorecards complete
    TurnMgr-->>UseCase: true
    deactivate TurnMgr

    UseCase->>TurnMgr: CalculateFinalRankings(room)
    activate TurnMgr
    TurnMgr->>Room: get all player scores
    TurnMgr->>TurnMgr: sort by total score
    TurnMgr-->>UseCase: rankings
    deactivate TurnMgr

    UseCase->>Room: FinishGame(rankings)
    activate Room
    Room->>Room: state = Finished
    Room->>Room: store rankings
    Room-->>UseCase: nil
    deactivate Room

    UseCase->>EventBus: Publish(GameFinishedEvent)
    activate EventBus
    EventBus-->>UseCase:
    deactivate EventBus

    UseCase-->>Player: output
    deactivate UseCase

    EventBus->>AllPlayers: GameFinishedEvent (broadcast)
```

---

## 6. 状態遷移図

### 6.1 Room State Transitions

```mermaid
stateDiagram-v2
    [*] --> WaitingForPlayers: Room Created

    WaitingForPlayers --> WaitingForPlayers: Player Joins
    WaitingForPlayers --> WaitingForPlayers: Player Leaves
    WaitingForPlayers --> InProgress: Start Game<br/>(>= 2 players)
    WaitingForPlayers --> Cancelled: Host Leaves

    InProgress --> InProgress: Roll Dice
    InProgress --> InProgress: Choose Category
    InProgress --> InProgress: Next Turn
    InProgress --> Finished: All Scorecards Complete
    InProgress --> Cancelled: Force Cancel

    Finished --> [*]
    Cancelled --> [*]

    note right of WaitingForPlayers
        - Can add/remove players
        - Can start game
        - Cannot play
    end note

    note right of InProgress
        - Cannot add/remove players
        - Can roll dice
        - Can choose category
        - Players can disconnect/reconnect
    end note

    note right of Finished
        - Read-only state
        - Display final rankings
        - Room can be deleted
    end note
```

### 6.2 Turn Phase Transitions

```mermaid
stateDiagram-v2
    [*] --> RollPhase: Turn Started

    RollPhase --> RollPhase: Roll Dice<br/>(rollCount < 3)
    RollPhase --> CategorySelection: Roll Dice<br/>(rollCount = 3)
    RollPhase --> CategorySelection: Skip Rolls

    CategorySelection --> Completed: Select Category

    Completed --> [*]: Turn Ended

    note right of RollPhase
        - Can roll dice (max 3 times)
        - Can hold/release dice
        - Can skip to category selection
    end note

    note right of CategorySelection
        - Must select category
        - Cannot roll anymore
        - Score calculated
    end note
```

### 6.3 Player State Transitions

```mermaid
stateDiagram-v2
    [*] --> Active: Player Joined

    Active --> Disconnected: Connection Lost
    Active --> [*]: Leave Room

    Disconnected --> Reconnecting: Reconnect Attempt
    Disconnected --> [*]: Timeout (5 min)

    Reconnecting --> Active: Reconnect Success
    Reconnecting --> Disconnected: Reconnect Failed

    note right of Active
        - Can send commands
        - Receives events
        - Turn timer active
    end note

    note right of Disconnected
        - Turn skipped
        - 5-minute timeout
        - Can reconnect
    end note
```

---

## 7. コンポーネント図

```mermaid
graph TB
    subgraph Client["Client Application"]
        CLI[CLI Interface]
        ClientConn[TCP Client]
    end

    subgraph Server["Server Application"]
        TCPServer[TCP Server]
        ConnMgr[Connection Manager]

        subgraph Presentation["Presentation Layer"]
            MsgRouter[Message Router]
            Handlers[Message Handlers]
        end

        subgraph Application["Application Layer"]
            UseCases[Use Cases]
            DTOs[DTOs]
        end

        subgraph Domain["Domain Layer"]
            Entities[Entities]
            Services[Domain Services]
            RepoInterfaces[Repository Interfaces]
        end

        subgraph Infrastructure["Infrastructure Layer"]
            Repos[Repositories]
            EventBus[Event Bus]
        end
    end

    CLI -->|JSON/TCP| ClientConn
    ClientConn -->|Messages| TCPServer
    TCPServer --> ConnMgr
    ConnMgr --> MsgRouter
    MsgRouter --> Handlers
    Handlers --> UseCases
    UseCases --> Services
    UseCases --> RepoInterfaces
    RepoInterfaces -.implements.-> Repos
    Repos --> Entities
    UseCases --> EventBus
    EventBus -->|Events| ConnMgr
```

---

## 8. データフロー図

```mermaid
flowchart LR
    subgraph Client
        A[User Input]
    end

    subgraph Presentation
        B[Handler]
        C[Validation]
    end

    subgraph Application
        D[Use Case]
        E[DTO Transform]
    end

    subgraph Domain
        F[Business Logic]
        G[Entity State]
    end

    subgraph Infrastructure
        H[Repository]
        I[Event Bus]
    end

    subgraph Response
        J[Success/Error]
        K[Events]
    end

    A -->|Command| B
    B -->|Validate| C
    C -->|Input| D
    D -->|Transform| E
    E -->|Domain Operation| F
    F -->|Update| G
    G -->|Persist| H
    H -->|Save| G
    D -->|Publish| I
    I -->|Broadcast| K
    D -->|Result| J
    J -->|Response| A
    K -->|Notify| Client
```

---

## まとめ

これらの図により：

✅ **視覚的理解**: システムの構造と振る舞いを直感的に把握
✅ **関係性の明確化**: クラス間、レイヤー間の関係が明確
✅ **フローの追跡**: リクエストの流れを時系列で追跡可能
✅ **状態管理の可視化**: 状態遷移のルールが明確
✅ **コミュニケーション**: チーム内での設計議論に活用可能

次のステップ: テスタビリティ設計書で、テスト戦略とモック設計を定義します。
