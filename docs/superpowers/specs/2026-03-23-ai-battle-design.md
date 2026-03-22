# AI Battle Design Spec

## Overview

AI Battle は、異なる戦術を持つAI同士を対戦させて観戦する機能。`yatz battle` コマンド一つで、ヤッツィーの戦術を比較検証できる。

**目的:**
- 戦術の強さを定量的に比較（Greedy vs Statistical vs LLM）
- LLM にペルソナを与えて対戦させるエンターテインメント
- 新しい戦術のベンチマーク環境

**ビジョン:** 戦術をプラグインとして追加するだけで、既存のバトルエンジン・観戦UIがそのまま使える拡張性の高い設計。

## Architecture

Strategy パターンによる戦術プラグイン設計。全てのAIプレイヤーは `Strategy` インターフェースを通じて意思決定する。

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐
│ yatz battle │────▶│ RunBattle()  │────▶│   AIPlayer    │
│  (CLI cmd)  │     │ (engine)     │     │ + Strategy    │
└─────────────┘     └──────┬───────┘     └───────┬───────┘
                           │                     │
                    OnTurnDone callback    DecideAction()
                           │                     │
                    ┌──────▼───────┐     ┌───────▼───────┐
                    │  Spectator   │     │  GreedyStrat  │
                    │  TUI / Quiet │     │  StatStrat    │
                    └──────────────┘     │  LLMStrat     │
                                         └───────────────┘
```

## Strategy Interface

`engine/strategy.go` で定義。全ての戦術はこのインターフェースを実装する。

```go
type TurnAction struct {
    Type     string   // "hold" or "score"
    Indices  []int    // hold: dice indices to keep
    Category Category // score: category to score in
}

type Strategy interface {
    Name() string
    DecideAction(dice [5]int, rollCount int, scorecard Scorecard, available []Category) TurnAction
}
```

`DecideAction` はターン中に複数回呼ばれる。`"hold"` を返すとダイスをキープして振り直し、`"score"` を返すとカテゴリにスコアして手番終了。3回目のロール後（`rollCount >= MaxRolls`）は必ず `"score"` を返す必要がある。

### Greedy Strategy (`engine/strategy_greedy.go`)

最もシンプルなベースライン戦術。

- 常に即座にスコアする（Hold を使わない）
- 利用可能なカテゴリから最高得点のものを選択
- `bestCategoryForDice()` で全カテゴリのスコアを計算し、最大値を返す

```go
type GreedyStrategy struct{}
func (s *GreedyStrategy) Name() string { return "greedy" }
```

### Statistical Strategy (`engine/strategy_statistical.go`)

期待値計算に基づく最適判断戦術。

- 3回目のロール後は Greedy と同様に即座にスコア
- それ以外のロールでは、全32通りのホールド組み合わせの期待値を計算
- 即座にスコアする場合の得点と比較し、期待値が上回るホールドがあれば採用
- 上段ボーナス（63点以上で35点追加）への接近度を加味した補正あり

**期待値計算 (`engine/expected_value.go`):**
- `holdCombinations()`: 全32通り（5ビットマスク）のホールドパターンを事前計算
- `expectedValue()`: フリーダイスの全出目パターンを列挙し、各パターンで最高スコアとなるカテゴリの得点を平均化
- `expectedValueWithBonus()`: 上段ボーナスに近い場合、期待値に `UpperBonusValue * 0.1` の補正を加える
- 計算量: O(6^freeDice × |available|) — 最大 6^5 = 7,776 パターン × 13 カテゴリ

```go
type StatisticalStrategy struct{}
func (s *StatisticalStrategy) Name() string { return "statistical" }
```

### LLM Strategy (`bot/strategy_llm.go`)

Claude API を直接呼び出すLLM戦術。ペルソナに応じたプレイスタイルを実現。

- `anthropic-sdk-go` で Claude API に直接リクエスト
- システムプロンプトにペルソナ情報（性格・戦略・口癖）を注入
- ユーザープロンプトに現在のダイス・ロール回数・スコアカード状態を提供
- レスポンスは JSON 形式（`{"action":"hold"|"score", "indices":[...], "category":"...", "reasoning":"..."}`）
- パースエラーや無効なアクション時は `GreedyStrategy` にフォールバック

```go
type LLMStrategy struct {
    client  anthropic.Client
    model   string
    persona *Persona
}
func (s *LLMStrategy) Name() string { return "llm:" + s.persona.Name }
```

**API Key 方針:** MCP 経由ではなく直接 API Key を使用。理由:
1. バトルは1ゲームで最大39回（13ターン × 最大3アクション）のAPI呼び出しが発生
2. MCP ラウンドトリップのオーバーヘッドを排除し、高速な対戦を実現
3. `--api-key` フラグまたは `ANTHROPIC_API_KEY` 環境変数で指定

## Persona System

Markdown ベースのキャラクター定義。`personas/` ディレクトリに配置。

### フォーマット (`bot/persona.go`)

```markdown
# キャラクター名
## 性格
性格の説明...

## 戦略
- 戦略ポイント1
- 戦略ポイント2

## 口癖
「セリフ」
```

**パーサー (`LoadPersona`):**
- `# ` で始まる行 → `Name`
- `## 性格` / `## personality` → `Personality`
- `## 戦略` / `## strategy` → `Strategy`
- `## 口癖` / `## catchphrase` → `Catchphrase`
- 名前が空の場合は `"LLM"` をデフォルト値とする

### 同梱ペルソナ

| ファイル | キャラクター | 特徴 |
|----------|-------------|------|
| `personas/aggressive.md` | アタッカー | ヤッツィー・ラージストレート最優先。リスク許容型 |
| `personas/defensive.md` | ディフェンダー | 上段ボーナス最優先。堅実・確実な得点積み上げ |
| `personas/gambler.md` | ギャンブラー | 常にヤッツィー狙い。直感重視、確率無視 |

## Battle Execution

### BattleConfig (`engine/battle.go`)

```go
type BattlePlayer struct {
    Name     string
    Strategy Strategy
}

type BattleConfig struct {
    Players    []BattlePlayer
    Seed       int64
    OnTurnDone func(result AITurnResult)
}
```

- `Players`: 2名以上のプレイヤー（各自に Strategy を割り当て）
- `Seed`: 乱数シード（0 = 現在時刻ベース）。再現性のあるテストに利用
- `OnTurnDone`: 各ターン完了時のコールバック。TUI への結果ストリーミングに使用

### ターン実行フロー

```
RunBattle(cfg)
  └─ NewGame(names, src)
  └─ for game.Phase != PhaseFinished:
       └─ ais[current].PlayTurn()
            └─ game.Roll()
            └─ loop:
            │    └─ strategy.DecideAction(dice, rollCount, scorecard, available)
            │    └─ "hold" → game.Hold(indices) → continue
            │    └─ "score" → game.Score(category) → break
            └─ return AITurnResult{PlayerName, Dice, Category, Score, StrategyName, HoldHistory}
       └─ cfg.OnTurnDone(result)
```

`AITurnResult` には `HoldHistory` が含まれ、各ロールでどのダイスをキープしたかを記録する。観戦UIでの思考過程の可視化に使用。

## TUI Spectator (`cli/spectator.go`)

Bubbletea v2 による観戦UI。チャネル経由でバトル結果をリアルタイム受信。

### アーキテクチャ

```
goroutine: RunBattle() ──▶ resultCh ──▶ spectatorModel.Update()
                                         ├─ specResultMsg → 表示更新
                                         ├─ specTickMsg   → 次の結果を取得
                                         └─ specDoneMsg   → ゲーム終了画面
```

### 画面構成

**観戦中 (`specWatching`):**
- ターン番号・プレイヤー名・戦術名
- ホールド履歴（キープしたダイスを `[*N*]` で強調表示）
- 最終ダイス・選択カテゴリ・得点
- リアルタイムスコアカード（全プレイヤー横並び）
- `speed` で設定した間隔で自動進行、任意キーでスキップ可能

**ゲーム終了 (`specGameOver`):**
- 完全なスコアカード（上段ボーナス表示あり）
- 勝者発表

### Quiet Mode

TUI を使わず、結果のみをテーブル形式で出力。複数ラウンドの統計比較に最適。

```
=== Battle Results (100 games) ===
Player            Wins   Avg Score   Max Score
Statistical        72       213.4         312
Greedy             28       187.2         289
```

## CLI Interface (`cmd/yatz/battle.go`)

### コマンド

```
yatz battle [flags]
```

### フラグ

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--players` | `Greedy:greedy,Statistical:statistical` | `Name:strategy` 形式のプレイヤー指定 |
| `--speed` | `1s` | ターン表示間隔 |
| `--seed` | `0` (ランダム) | 乱数シード |
| `--rounds` | `1` | 連続対戦数 |
| `--quiet` | `false` | TUI なし、結果のみ表示 |
| `--api-key` | `""` | Claude API Key（`ANTHROPIC_API_KEY` 環境変数も可） |
| `--model` | `claude-haiku-4-5-20251001` | LLM 戦術用モデル |

### 使用例

```bash
# 基本: Greedy vs Statistical
yatz battle

# LLM ペルソナ対戦
yatz battle --players "Attacker:llm:personas/aggressive.md,Defender:llm:personas/defensive.md"

# 100回対戦の統計
yatz battle --rounds 100 --quiet

# 3者対戦（シード固定）
yatz battle --players "G:greedy,S:statistical,L:llm" --seed 42

# 表示速度を高速化
yatz battle --speed 200ms
```

### Strategy 指定形式

| 指定 | 説明 |
|------|------|
| `greedy` | GreedyStrategy |
| `statistical` | StatisticalStrategy |
| `llm` | LLMStrategy（デフォルトペルソナ） |
| `llm:<path>` | LLMStrategy（カスタムペルソナ） |
