# YatzCLI v2 Rebuild Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild YatzCLI as a single binary with local AI play (MCP), interactive CLI, and P2P online play.

**Architecture:** State-machine game engine (`engine/`) with `GameClient` interface abstracting local vs remote access. MCP server for LLM integration, P2P host-authority model for online play, serverless matchmaking via Lambda.

**Tech Stack:** Go 1.22+, cobra, bubbletea/huh, mcp-go, aws-lambda-go, goreleaser

**Spec:** `docs/superpowers/specs/2026-03-22-yatzcli-rebuild-design.md`

---

## Phase 0: Project Setup

### Task 0: Clean slate and initialize new module structure

**Files:**
- Remove: all files under `cmd/`, `client/`, `server/`, `game/`, `messages/`, `network/`
- Create: `cmd/yatz/main.go`
- Modify: `go.mod`

- [ ] **Step 1: Create a new branch for the rebuild**

```bash
git checkout -b feature/v2-rebuild
```

- [ ] **Step 2: Remove old source code**

Remove old packages. Keep `docs/`, `README.md`, `go.mod`, `go.sum`, `.gitignore`.

```bash
rm -rf cmd/ client/ server/ game/ messages/ network/
```

- [ ] **Step 3: Update go.mod to Go 1.22+**

Update `go.mod` module directive to `go 1.22`. Remove old dependencies (they'll be re-added as needed).

```bash
# Reset go.mod to just the module declaration
cat > go.mod << 'GOMOD'
module github.com/edge2992/yatzcli

go 1.22.0
GOMOD
rm -f go.sum
```

- [ ] **Step 4: Create minimal main.go with cobra**

```bash
mkdir -p cmd/yatz
```

Create `cmd/yatz/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "yatz",
	Short: "Yahtzee CLI game",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Install dependencies and verify build**

```bash
go get github.com/spf13/cobra
go mod tidy
go build ./cmd/yatz/
```

Expected: binary builds successfully.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "chore: clean old code and initialize v2 project structure"
```

---

## Phase 1: Game Engine

### Task 1: Category and Scorecard types

**Files:**
- Create: `engine/category.go`
- Create: `engine/scorecard.go`
- Create: `engine/scorecard_test.go`

- [ ] **Step 1: Write category type and constants**

Create `engine/category.go`:

```go
package engine

type Category string

const (
	Ones         Category = "ones"
	Twos         Category = "twos"
	Threes       Category = "threes"
	Fours        Category = "fours"
	Fives        Category = "fives"
	Sixes        Category = "sixes"
	ThreeOfAKind Category = "three_of_a_kind"
	FourOfAKind  Category = "four_of_a_kind"
	FullHouse    Category = "full_house"
	SmallStraight Category = "small_straight"
	LargeStraight Category = "large_straight"
	Yahtzee      Category = "yahtzee"
	Chance       Category = "chance"
)

var AllCategories = []Category{
	Ones, Twos, Threes, Fours, Fives, Sixes,
	ThreeOfAKind, FourOfAKind, FullHouse,
	SmallStraight, LargeStraight, Yahtzee, Chance,
}

var UpperCategories = []Category{Ones, Twos, Threes, Fours, Fives, Sixes}

const UpperBonusThreshold = 63
const UpperBonusValue = 35
```

- [ ] **Step 2: Write failing tests for Scorecard**

Create `engine/scorecard_test.go`:

```go
package engine

import "testing"

func TestNewScorecard(t *testing.T) {
	sc := NewScorecard()
	for _, c := range AllCategories {
		if sc.IsFilled(c) {
			t.Errorf("category %s should not be filled", c)
		}
	}
}

func TestScorecard_Fill(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)
	if !sc.IsFilled(Ones) {
		t.Error("Ones should be filled")
	}
	if sc.GetScore(Ones) != 3 {
		t.Errorf("expected 3, got %d", sc.GetScore(Ones))
	}
}

func TestScorecard_FillZero(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(FullHouse, 0)
	if !sc.IsFilled(FullHouse) {
		t.Error("FullHouse should be filled even with 0")
	}
	if sc.GetScore(FullHouse) != 0 {
		t.Errorf("expected 0, got %d", sc.GetScore(FullHouse))
	}
}

func TestScorecard_AvailableCategories(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)
	avail := sc.AvailableCategories()
	for _, c := range avail {
		if c == Ones {
			t.Error("Ones should not be available after filling")
		}
	}
	if len(avail) != len(AllCategories)-1 {
		t.Errorf("expected %d available, got %d", len(AllCategories)-1, len(avail))
	}
}

func TestScorecard_UpperBonus(t *testing.T) {
	sc := NewScorecard()
	// Fill upper section to exactly 63 (3*1 + 3*2 + 3*3 + 3*4 + 3*5 + 3*6 = 63)
	sc.Fill(Ones, 3)
	sc.Fill(Twos, 6)
	sc.Fill(Threes, 9)
	sc.Fill(Fours, 12)
	sc.Fill(Fives, 15)
	sc.Fill(Sixes, 18)
	if sc.UpperTotal() != 63 {
		t.Errorf("expected upper total 63, got %d", sc.UpperTotal())
	}
	if !sc.HasUpperBonus() {
		t.Error("should have upper bonus at 63")
	}
	if sc.Total() != 63+UpperBonusValue {
		t.Errorf("expected total %d, got %d", 63+UpperBonusValue, sc.Total())
	}
}

func TestScorecard_NoUpperBonus(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 2)
	sc.Fill(Twos, 4)
	sc.Fill(Threes, 6)
	sc.Fill(Fours, 8)
	sc.Fill(Fives, 10)
	sc.Fill(Sixes, 12)
	if sc.UpperTotal() != 42 {
		t.Errorf("expected upper total 42, got %d", sc.UpperTotal())
	}
	if sc.HasUpperBonus() {
		t.Error("should not have upper bonus at 42")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./engine/ -v
```

Expected: FAIL — `NewScorecard` not defined.

- [ ] **Step 4: Implement Scorecard**

Create `engine/scorecard.go`:

```go
package engine

type Scorecard struct {
	scores map[Category]*int
}

func NewScorecard() Scorecard {
	return Scorecard{scores: make(map[Category]*int)}
}

func (sc *Scorecard) Fill(c Category, score int) {
	s := score
	sc.scores[c] = &s
}

func (sc *Scorecard) IsFilled(c Category) bool {
	return sc.scores[c] != nil
}

func (sc *Scorecard) GetScore(c Category) int {
	if sc.scores[c] == nil {
		return 0
	}
	return *sc.scores[c]
}

func (sc *Scorecard) AvailableCategories() []Category {
	var avail []Category
	for _, c := range AllCategories {
		if !sc.IsFilled(c) {
			avail = append(avail, c)
		}
	}
	return avail
}

func (sc *Scorecard) UpperTotal() int {
	total := 0
	for _, c := range UpperCategories {
		total += sc.GetScore(c)
	}
	return total
}

func (sc *Scorecard) HasUpperBonus() bool {
	return sc.UpperTotal() >= UpperBonusThreshold
}

func (sc *Scorecard) Total() int {
	total := 0
	for _, c := range AllCategories {
		total += sc.GetScore(c)
	}
	if sc.HasUpperBonus() {
		total += UpperBonusValue
	}
	return total
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./engine/ -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add engine/
git commit -m "feat(engine): add Category type and Scorecard with upper bonus"
```

---

### Task 2: Score calculation functions

**Files:**
- Create: `engine/scoring.go`
- Create: `engine/scoring_test.go`

- [ ] **Step 1: Write failing tests for all scoring categories**

Create `engine/scoring_test.go`:

```go
package engine

import "testing"

func TestCalcScore_Upper(t *testing.T) {
	dice := [5]int{1, 1, 3, 1, 5}
	if got := CalcScore(Ones, dice); got != 3 {
		t.Errorf("Ones: expected 3, got %d", got)
	}
	if got := CalcScore(Threes, dice); got != 3 {
		t.Errorf("Threes: expected 3, got %d", got)
	}
	if got := CalcScore(Twos, dice); got != 0 {
		t.Errorf("Twos: expected 0, got %d", got)
	}
}

func TestCalcScore_ThreeOfAKind(t *testing.T) {
	if got := CalcScore(ThreeOfAKind, [5]int{3, 3, 3, 4, 5}); got != 18 {
		t.Errorf("expected 18, got %d", got)
	}
	if got := CalcScore(ThreeOfAKind, [5]int{1, 2, 3, 4, 5}); got != 0 {
		t.Errorf("no three of a kind: expected 0, got %d", got)
	}
}

func TestCalcScore_FourOfAKind(t *testing.T) {
	if got := CalcScore(FourOfAKind, [5]int{2, 2, 2, 2, 5}); got != 13 {
		t.Errorf("expected 13, got %d", got)
	}
	if got := CalcScore(FourOfAKind, [5]int{2, 2, 2, 3, 5}); got != 0 {
		t.Errorf("no four of a kind: expected 0, got %d", got)
	}
}

func TestCalcScore_FullHouse(t *testing.T) {
	if got := CalcScore(FullHouse, [5]int{3, 3, 5, 5, 5}); got != 25 {
		t.Errorf("expected 25, got %d", got)
	}
	if got := CalcScore(FullHouse, [5]int{3, 3, 3, 5, 5}); got != 25 {
		t.Errorf("expected 25, got %d", got)
	}
	if got := CalcScore(FullHouse, [5]int{1, 2, 3, 4, 5}); got != 0 {
		t.Errorf("no full house: expected 0, got %d", got)
	}
	// Yahtzee is NOT a full house
	if got := CalcScore(FullHouse, [5]int{4, 4, 4, 4, 4}); got != 0 {
		t.Errorf("yahtzee is not full house: expected 0, got %d", got)
	}
}

func TestCalcScore_SmallStraight(t *testing.T) {
	if got := CalcScore(SmallStraight, [5]int{1, 2, 3, 4, 6}); got != 30 {
		t.Errorf("expected 30, got %d", got)
	}
	if got := CalcScore(SmallStraight, [5]int{2, 3, 4, 5, 5}); got != 30 {
		t.Errorf("expected 30, got %d", got)
	}
	if got := CalcScore(SmallStraight, [5]int{1, 2, 4, 5, 6}); got != 0 {
		t.Errorf("no small straight: expected 0, got %d", got)
	}
}

func TestCalcScore_LargeStraight(t *testing.T) {
	if got := CalcScore(LargeStraight, [5]int{1, 2, 3, 4, 5}); got != 40 {
		t.Errorf("expected 40, got %d", got)
	}
	if got := CalcScore(LargeStraight, [5]int{2, 3, 4, 5, 6}); got != 40 {
		t.Errorf("expected 40, got %d", got)
	}
	if got := CalcScore(LargeStraight, [5]int{1, 2, 3, 4, 6}); got != 0 {
		t.Errorf("no large straight: expected 0, got %d", got)
	}
}

func TestCalcScore_Yahtzee(t *testing.T) {
	if got := CalcScore(Yahtzee, [5]int{4, 4, 4, 4, 4}); got != 50 {
		t.Errorf("expected 50, got %d", got)
	}
	if got := CalcScore(Yahtzee, [5]int{4, 4, 4, 4, 5}); got != 0 {
		t.Errorf("no yahtzee: expected 0, got %d", got)
	}
}

func TestCalcScore_Chance(t *testing.T) {
	if got := CalcScore(Chance, [5]int{1, 2, 3, 4, 5}); got != 15 {
		t.Errorf("expected 15, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./engine/ -run TestCalcScore -v
```

Expected: FAIL — `CalcScore` not defined.

- [ ] **Step 3: Implement CalcScore**

Create `engine/scoring.go`:

```go
package engine

import "sort"

func CalcScore(c Category, dice [5]int) int {
	switch c {
	case Ones:
		return countValue(dice, 1)
	case Twos:
		return countValue(dice, 2) * 2
	case Threes:
		return countValue(dice, 3) * 3
	case Fours:
		return countValue(dice, 4) * 4
	case Fives:
		return countValue(dice, 5) * 5
	case Sixes:
		return countValue(dice, 6) * 6
	case ThreeOfAKind:
		if hasNOfAKind(dice, 3) {
			return sum(dice)
		}
		return 0
	case FourOfAKind:
		if hasNOfAKind(dice, 4) {
			return sum(dice)
		}
		return 0
	case FullHouse:
		if isFullHouse(dice) {
			return 25
		}
		return 0
	case SmallStraight:
		if hasStraight(dice, 4) {
			return 30
		}
		return 0
	case LargeStraight:
		if hasStraight(dice, 5) {
			return 40
		}
		return 0
	case Yahtzee:
		if hasNOfAKind(dice, 5) {
			return 50
		}
		return 0
	case Chance:
		return sum(dice)
	}
	return 0
}

func countValue(dice [5]int, val int) int {
	count := 0
	for _, d := range dice {
		if d == val {
			count++
		}
	}
	return count
}

func sum(dice [5]int) int {
	s := 0
	for _, d := range dice {
		s += d
	}
	return s
}

func counts(dice [5]int) map[int]int {
	m := make(map[int]int)
	for _, d := range dice {
		m[d]++
	}
	return m
}

func hasNOfAKind(dice [5]int, n int) bool {
	for _, c := range counts(dice) {
		if c >= n {
			return true
		}
	}
	return false
}

func isFullHouse(dice [5]int) bool {
	c := counts(dice)
	if len(c) != 2 {
		return false
	}
	for _, v := range c {
		if v == 2 || v == 3 {
			return true
		}
	}
	return false
}

func hasStraight(dice [5]int, length int) bool {
	sorted := make([]int, len(dice))
	copy(sorted, dice[:])
	sort.Ints(sorted)
	// Deduplicate
	unique := []int{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			unique = append(unique, sorted[i])
		}
	}
	if len(unique) < length {
		return false
	}
	run := 1
	for i := 1; i < len(unique); i++ {
		if unique[i] == unique[i-1]+1 {
			run++
			if run >= length {
				return true
			}
		} else {
			run = 1
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./engine/ -run TestCalcScore -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/scoring.go engine/scoring_test.go
git commit -m "feat(engine): add score calculation for all categories"
```

---

### Task 3: Dice rolling

**Files:**
- Create: `engine/dice.go`
- Create: `engine/dice_test.go`

- [ ] **Step 1: Write failing tests for dice operations**

Create `engine/dice_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

func TestRollAll(t *testing.T) {
	src := rand.NewSource(42)
	dice := RollAll(src)
	for i, d := range dice {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, d)
		}
	}
}

func TestRollAll_Deterministic(t *testing.T) {
	d1 := RollAll(rand.NewSource(42))
	d2 := RollAll(rand.NewSource(42))
	if d1 != d2 {
		t.Errorf("same seed should produce same dice: %v != %v", d1, d2)
	}
}

func TestReroll(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	hold := []int{0, 2, 4} // hold indices 0, 2, 4 (values 1, 3, 5)
	result := Reroll(original, hold, src)
	// Held dice should not change
	if result[0] != 1 || result[2] != 3 || result[4] != 5 {
		t.Errorf("held dice changed: %v", result)
	}
	// Unheld dice should be in range (might happen to be same value, that's ok)
	for _, i := range []int{1, 3} {
		if result[i] < 1 || result[i] > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, result[i])
		}
	}
}

func TestReroll_HoldAll(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	hold := []int{0, 1, 2, 3, 4}
	result := Reroll(original, hold, src)
	if result != original {
		t.Errorf("holding all should not change dice: %v != %v", result, original)
	}
}

func TestReroll_HoldNone(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	result := Reroll(original, nil, src)
	for i, d := range result {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, d)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./engine/ -run TestRoll -v
```

Expected: FAIL — `RollAll` not defined.

- [ ] **Step 3: Implement dice operations**

Create `engine/dice.go`:

```go
package engine

import "math/rand"

func RollAll(src rand.Source) [5]int {
	r := rand.New(src)
	var dice [5]int
	for i := range dice {
		dice[i] = r.Intn(6) + 1
	}
	return dice
}

func Reroll(dice [5]int, hold []int, src rand.Source) [5]int {
	holdSet := make(map[int]bool)
	for _, i := range hold {
		holdSet[i] = true
	}
	r := rand.New(src)
	var result [5]int
	for i := range dice {
		if holdSet[i] {
			result[i] = dice[i]
		} else {
			result[i] = r.Intn(6) + 1
		}
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./engine/ -run TestRoll -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/dice.go engine/dice_test.go
git commit -m "feat(engine): add dice rolling with deterministic RNG support"
```

---

### Task 4: Game state machine

**Files:**
- Create: `engine/game.go`
- Create: `engine/game_test.go`

- [ ] **Step 1: Write failing tests for game state machine**

Create `engine/game_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

func newTestGame() *Game {
	return NewGame([]string{"Alice", "Bob"}, rand.NewSource(42))
}

func TestNewGame(t *testing.T) {
	g := newTestGame()
	if g.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", g.Phase)
	}
	if len(g.Players) != 2 {
		t.Errorf("expected 2 players, got %d", len(g.Players))
	}
	if g.Round != 1 {
		t.Errorf("expected round 1, got %d", g.Round)
	}
	if g.Current != 0 {
		t.Errorf("expected current 0, got %d", g.Current)
	}
}

func TestGame_Roll(t *testing.T) {
	g := newTestGame()
	err := g.Roll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.RollCount != 1 {
		t.Errorf("expected RollCount 1, got %d", g.RollCount)
	}
	for i, d := range g.Dice {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, d)
		}
	}
}

func TestGame_Roll_NotInitialRoll(t *testing.T) {
	g := newTestGame()
	g.Roll()
	err := g.Roll()
	if err == nil {
		t.Error("expected error for second Roll() call")
	}
}

func TestGame_Hold(t *testing.T) {
	g := newTestGame()
	g.Roll()
	original := g.Dice
	err := g.Hold([]int{0, 2, 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.RollCount != 2 {
		t.Errorf("expected RollCount 2, got %d", g.RollCount)
	}
	if g.Dice[0] != original[0] || g.Dice[2] != original[2] || g.Dice[4] != original[4] {
		t.Error("held dice should not change")
	}
}

func TestGame_Hold_BeforeRoll(t *testing.T) {
	g := newTestGame()
	err := g.Hold([]int{0})
	if err == nil {
		t.Error("expected error for Hold before initial Roll")
	}
}

func TestGame_Hold_MaxRolls(t *testing.T) {
	g := newTestGame()
	g.Roll()       // rollCount=1
	g.Hold([]int{}) // rollCount=2
	g.Hold([]int{}) // rollCount=3 -> PhaseChoosing
	if g.Phase != PhaseChoosing {
		t.Errorf("expected PhaseChoosing after 3 rolls, got %d", g.Phase)
	}
	err := g.Hold([]int{})
	if err == nil {
		t.Error("expected error for Hold in choosing phase")
	}
}

func TestGame_Roll_AfterMaxRolls(t *testing.T) {
	g := newTestGame()
	g.Roll()
	g.Hold([]int{})
	g.Hold([]int{})
	err := g.Roll()
	if err == nil {
		t.Error("expected error for Roll in choosing phase")
	}
}

func TestGame_Score(t *testing.T) {
	g := newTestGame()
	g.Roll()
	err := g.Score(Chance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !g.Players[0].Scorecard.IsFilled(Chance) {
		t.Error("Chance should be filled")
	}
	// After scoring, turn passes to next player
	if g.Current != 1 {
		t.Errorf("expected current 1, got %d", g.Current)
	}
	if g.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling for next player, got %d", g.Phase)
	}
	if g.RollCount != 0 {
		t.Errorf("expected RollCount 0, got %d", g.RollCount)
	}
}

func TestGame_Score_BeforeRoll(t *testing.T) {
	g := newTestGame()
	err := g.Score(Chance)
	if err == nil {
		t.Error("expected error for Score before Roll")
	}
}

func TestGame_Score_AlreadyFilled(t *testing.T) {
	g := newTestGame()
	g.Roll()
	g.Score(Chance)
	// Player 2's turn
	g.Roll()
	g.Score(Ones)
	// Back to Player 1, round 2
	g.Roll()
	err := g.Score(Chance)
	if err == nil {
		t.Error("expected error for already filled category")
	}
}

func TestGame_FullGame(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	categories := AllCategories
	for i := 0; i < 13; i++ {
		if g.Phase == PhaseFinished {
			t.Fatalf("game ended too early at round %d", i+1)
		}
		g.Roll()
		g.Score(categories[i])
	}
	if g.Phase != PhaseFinished {
		t.Error("game should be finished after 13 rounds")
	}
}

func TestGame_GetState(t *testing.T) {
	g := newTestGame()
	g.Roll()
	state := g.GetState()
	if state.Phase != PhaseRolling {
		t.Errorf("expected PhaseRolling, got %d", state.Phase)
	}
	if state.CurrentPlayer != g.Players[0].Name {
		t.Errorf("expected Alice, got %s", state.CurrentPlayer)
	}
}

func TestGame_GetAvailableCategories(t *testing.T) {
	g := newTestGame()
	g.Roll()
	avail := g.GetAvailableCategories()
	if len(avail) != 13 {
		t.Errorf("expected 13 available, got %d", len(avail))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./engine/ -run TestGame -v -count=1
```

Expected: FAIL — `NewGame` not defined.

- [ ] **Step 3: Implement Game state machine**

Create `engine/game.go`:

```go
package engine

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type GamePhase int

const (
	PhaseWaiting  GamePhase = iota // Before game starts (unused in v1, reserved)
	PhaseRolling                   // Player can Roll() or Hold()
	PhaseChoosing                  // Player must Score() (after 3 rolls)
	PhaseFinished                  // Game over
)

const MaxRolls = 3
const MaxRounds = 13

// Game methods are NOT safe for concurrent use. Callers must synchronize externally.
type Game struct {
	Players   []Player
	Current   int
	Round     int
	Dice      [5]int
	RollCount int
	Phase     GamePhase
	rng       rand.Source
}

type Player struct {
	ID        string
	Name      string
	Scorecard Scorecard
}

type GameState struct {
	Players            []PlayerState
	CurrentPlayer      string
	CurrentPlayerIndex int
	Round              int
	Dice               [5]int
	RollCount          int
	Phase              GamePhase
	AvailableCategories []Category
}

type PlayerState struct {
	ID        string
	Name      string
	Scorecard Scorecard
}

func NewGame(playerNames []string, src rand.Source) *Game {
	if src == nil {
		src = rand.NewSource(time.Now().UnixNano())
	}
	players := make([]Player, len(playerNames))
	for i, name := range playerNames {
		players[i] = Player{
			ID:        fmt.Sprintf("player_%d", i),
			Name:      name,
			Scorecard: NewScorecard(),
		}
	}
	return &Game{
		Players: players,
		Current: 0,
		Round:   1,
		Phase:   PhaseRolling,
		rng:     src,
	}
}

func (g *Game) Roll() error {
	if g.Phase != PhaseRolling {
		return errors.New("cannot roll: game is not in rolling phase")
	}
	if g.RollCount != 0 {
		return errors.New("cannot roll: use Hold() for subsequent rolls")
	}
	g.Dice = RollAll(g.rng)
	g.RollCount = 1
	if g.RollCount >= MaxRolls {
		g.Phase = PhaseChoosing
	}
	return nil
}

func (g *Game) Hold(indices []int) error {
	if g.Phase != PhaseRolling {
		return errors.New("cannot hold: game is not in rolling phase")
	}
	if g.RollCount == 0 {
		return errors.New("cannot hold: must Roll() first")
	}
	if g.RollCount >= MaxRolls {
		return errors.New("cannot hold: no rolls remaining")
	}
	for _, i := range indices {
		if i < 0 || i > 4 {
			return fmt.Errorf("invalid hold index: %d", i)
		}
	}
	g.Dice = Reroll(g.Dice, indices, g.rng)
	g.RollCount++
	if g.RollCount >= MaxRolls {
		g.Phase = PhaseChoosing
	}
	return nil
}

func (g *Game) Score(category Category) error {
	if g.Phase != PhaseRolling && g.Phase != PhaseChoosing {
		return errors.New("cannot score: must be in rolling or choosing phase")
	}
	if g.RollCount == 0 {
		return errors.New("cannot score: must Roll() first")
	}
	player := &g.Players[g.Current]
	if player.Scorecard.IsFilled(category) {
		return fmt.Errorf("category %s is already filled", category)
	}
	score := CalcScore(category, g.Dice)
	player.Scorecard.Fill(category, score)
	g.advanceTurn()
	return nil
}

func (g *Game) advanceTurn() {
	g.Current++
	if g.Current >= len(g.Players) {
		g.Current = 0
		g.Round++
	}
	if g.Round > MaxRounds {
		g.Phase = PhaseFinished
		return
	}
	g.Phase = PhaseRolling
	g.RollCount = 0
	g.Dice = [5]int{}
}

func (g *Game) GetState() GameState {
	players := make([]PlayerState, len(g.Players))
	for i, p := range g.Players {
		players[i] = PlayerState{ID: p.ID, Name: p.Name, Scorecard: p.Scorecard}
	}
	var currentPlayer string
	if g.Phase != PhaseFinished {
		currentPlayer = g.Players[g.Current].Name
	}
	return GameState{
		Players:            players,
		CurrentPlayer:      currentPlayer,
		CurrentPlayerIndex: g.Current,
		Round:              g.Round,
		Dice:               g.Dice,
		RollCount:          g.RollCount,
		Phase:              g.Phase,
		AvailableCategories: g.GetAvailableCategories(),
	}
}

func (g *Game) GetAvailableCategories() []Category {
	if g.Phase == PhaseFinished {
		return nil
	}
	return g.Players[g.Current].Scorecard.AvailableCategories()
}

func (g *Game) GetScorecard(playerID string) *Scorecard {
	for i := range g.Players {
		if g.Players[i].ID == playerID {
			return &g.Players[i].Scorecard
		}
	}
	return nil
}
```

- [ ] **Step 4: Run all engine tests**

```bash
go test ./engine/ -v -count=1
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/game.go engine/game_test.go
git commit -m "feat(engine): add game state machine with turn management"
```

---

## Phase 2: GameClient + CLI Play

### Task 5: GameClient interface and LocalClient

**Files:**
- Create: `engine/client.go`
- Create: `engine/client_test.go`

- [ ] **Step 1: Write failing tests for LocalClient**

Create `engine/client_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

func TestLocalClient_Roll(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player_0", nil)
	state, err := c.Roll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.RollCount != 1 {
		t.Errorf("expected RollCount 1, got %d", state.RollCount)
	}
}

func TestLocalClient_Hold(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player_0", nil)
	c.Roll()
	state, err := c.Hold([]int{0, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.RollCount != 2 {
		t.Errorf("expected RollCount 2, got %d", state.RollCount)
	}
}

func TestLocalClient_Score(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player_0", nil)
	c.Roll()
	state, err := c.Score(Chance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.Players[0].Scorecard.IsFilled(Chance) {
		t.Error("Chance should be filled")
	}
}

func TestLocalClient_ScoreTriggersAI(t *testing.T) {
	g := NewGame([]string{"Alice", "Bot"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player_1")
	c := NewLocalClient(g, "player_0", []*AIPlayer{ai})
	c.Roll()
	state, err := c.Score(Chance)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After scoring, AI should have played and it's back to Alice
	if state.CurrentPlayer != "Alice" {
		t.Errorf("expected Alice's turn after AI auto-play, got %s", state.CurrentPlayer)
	}
	if state.Round != 2 {
		t.Errorf("expected round 2, got %d", state.Round)
	}
}

func TestLocalClient_GetState(t *testing.T) {
	g := NewGame([]string{"Alice"}, rand.NewSource(42))
	c := NewLocalClient(g, "player_0", nil)
	state, err := c.GetState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.CurrentPlayer != "Alice" {
		t.Errorf("expected Alice, got %s", state.CurrentPlayer)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./engine/ -run TestLocalClient -v
```

Expected: FAIL — `NewLocalClient` not defined.

- [ ] **Step 3: Implement GameClient interface and LocalClient**

Create `engine/client.go`:

```go
package engine

type GameClient interface {
	Roll() (*GameState, error)
	Hold(indices []int) (*GameState, error)
	Score(category Category) (*GameState, error)
	GetState() (*GameState, error)
}

type LocalClient struct {
	game     *Game
	playerID string
	ais      []*AIPlayer // AI opponents to auto-play after human actions
}

func NewLocalClient(game *Game, playerID string, ais []*AIPlayer) *LocalClient {
	return &LocalClient{game: game, playerID: playerID, ais: ais}
}

// runAITurns auto-plays all consecutive AI turns after a human action.
func (c *LocalClient) runAITurns() error {
	for c.game.Phase != PhaseFinished {
		current := c.game.Players[c.game.Current]
		if current.ID == c.playerID {
			break // Back to human's turn
		}
		for _, ai := range c.ais {
			if ai.playerID == current.ID {
				if err := ai.PlayTurn(); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (c *LocalClient) Roll() (*GameState, error) {
	if err := c.game.Roll(); err != nil {
		return nil, err
	}
	state := c.game.GetState()
	return &state, nil
}

func (c *LocalClient) Hold(indices []int) (*GameState, error) {
	if err := c.game.Hold(indices); err != nil {
		return nil, err
	}
	state := c.game.GetState()
	return &state, nil
}

func (c *LocalClient) Score(category Category) (*GameState, error) {
	if err := c.game.Score(category); err != nil {
		return nil, err
	}
	// After human scores, auto-play AI turns
	if err := c.runAITurns(); err != nil {
		return nil, err
	}
	state := c.game.GetState()
	return &state, nil
}

func (c *LocalClient) GetState() (*GameState, error) {
	state := c.game.GetState()
	return &state, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./engine/ -run TestLocalClient -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/client.go engine/client_test.go
git commit -m "feat(engine): add GameClient interface and LocalClient"
```

---

### Task 6: AI opponent

**Files:**
- Create: `engine/ai.go`
- Create: `engine/ai_test.go`

- [ ] **Step 1: Write failing tests for AI player**

Create `engine/ai_test.go`:

```go
package engine

import (
	"math/rand"
	"testing"
)

func TestAIPlayer_PlayTurn(t *testing.T) {
	g := NewGame([]string{"Human", "AI"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player_1")

	// Human plays turn
	g.Roll()
	g.Score(Chance)

	// AI plays its turn
	if g.Players[g.Current].ID != "player_1" {
		t.Fatal("expected AI's turn")
	}
	err := ai.PlayTurn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After AI plays, should be back to human
	if g.Current != 0 {
		t.Errorf("expected current 0 after AI turn, got %d", g.Current)
	}
}

func TestAIPlayer_PlayTurn_NotMyTurn(t *testing.T) {
	g := NewGame([]string{"Human", "AI"}, rand.NewSource(42))
	ai := NewAIPlayer(g, "player_1")
	err := ai.PlayTurn()
	if err == nil {
		t.Error("expected error when not AI's turn")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./engine/ -run TestAIPlayer -v
```

Expected: FAIL — `NewAIPlayer` not defined.

- [ ] **Step 3: Implement AI player (greedy strategy)**

Create `engine/ai.go`:

```go
package engine

import "errors"

type AIPlayer struct {
	game     *Game
	playerID string
}

func NewAIPlayer(game *Game, playerID string) *AIPlayer {
	return &AIPlayer{game: game, playerID: playerID}
}

func (ai *AIPlayer) PlayTurn() error {
	if ai.game.Players[ai.game.Current].ID != ai.playerID {
		return errors.New("not AI's turn")
	}
	if err := ai.game.Roll(); err != nil {
		return err
	}
	// Greedy: pick the category with the highest score
	best := ai.bestCategory()
	return ai.game.Score(best)
}

func (ai *AIPlayer) bestCategory() Category {
	avail := ai.game.GetAvailableCategories()
	bestScore := -1
	bestCat := avail[0]
	for _, c := range avail {
		score := CalcScore(c, ai.game.Dice)
		if score > bestScore {
			bestScore = score
			bestCat = c
		}
	}
	return bestCat
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./engine/ -run TestAIPlayer -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add engine/ai.go engine/ai_test.go
git commit -m "feat(engine): add greedy AI player"
```

---

### Task 7: CLI play command with bubbletea TUI

**Files:**
- Create: `cli/ui.go`
- Create: `cli/model.go`
- Modify: `cmd/yatz/main.go`

- [ ] **Step 1: Install TUI dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/olekukonko/tablewriter
```

- [ ] **Step 2: Create TUI model**

Create `cli/model.go` — The bubbletea model that drives the interactive game loop. This handles rendering the game state (dice, scorecard, available categories) and processing user input (roll, hold dice, select category).

The model wraps a `GameClient` interface so it works for both local and P2P games. Key states: `stateRolling` (user can roll/hold), `stateChoosing` (user selects a scoring category), `stateGameOver` (final scores displayed).

This is a large file (~200 lines) implementing `tea.Model` interface (Init, Update, View). Implementation should use lipgloss for styling and tablewriter for scorecard display.

- [ ] **Step 3: Create UI entry point**

Create `cli/ui.go`:

```go
package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/edge2992/yatzcli/engine"
)

func RunGame(client engine.GameClient, playerName string) error {
	m := newModel(client, playerName)
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Wire up `yatz play` command**

Update `cmd/yatz/main.go` to add the `play` subcommand:

```go
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play a local game against AI",
	RunE: func(cmd *cobra.Command, args []string) error {
		opponents, _ := cmd.Flags().GetInt("opponents")
		playerName, _ := cmd.Flags().GetString("name")

		names := []string{playerName}
		for i := 0; i < opponents; i++ {
			names = append(names, fmt.Sprintf("AI_%d", i+1))
		}

		game := engine.NewGame(names, nil)

		// Create AI players
		var ais []*engine.AIPlayer
		for i := 1; i <= opponents; i++ {
			ais = append(ais, engine.NewAIPlayer(game, fmt.Sprintf("player_%d", i)))
		}

		client := engine.NewLocalClient(game, "player_0", ais)
		return cli.RunGame(client, playerName)
	},
}

func init() {
	playCmd.Flags().IntP("opponents", "o", 1, "Number of AI opponents (1-3)")
	playCmd.Flags().StringP("name", "n", "Player", "Your player name")
	rootCmd.AddCommand(playCmd)
}
```

- [ ] **Step 5: Build and manual test**

```bash
go build ./cmd/yatz/ && ./yatz play
```

Expected: TUI launches, can play through a game.

- [ ] **Step 6: Commit**

```bash
git add cli/ cmd/
git commit -m "feat(cli): add interactive TUI play command with bubbletea"
```

---

## Phase 3: MCP Server

### Task 8: MCP server implementation

**Files:**
- Create: `mcp/server.go`
- Create: `mcp/server_test.go`
- Modify: `cmd/yatz/main.go`

- [ ] **Step 1: Install mcp-go dependency**

Check the latest API from Context7 MCP for `mark3labs/mcp-go` before implementation.

```bash
go get github.com/mark3labs/mcp-go
```

- [ ] **Step 2: Write failing tests for MCP tool handlers**

Create `mcp/server_test.go` — Tests for each MCP tool handler function (`handleNewGame`, `handleRollDice`, `handleHoldDice`, `handleScore`, `handleGetState`, `handleGetScorecard`). Each handler test creates a game with a deterministic RNG, calls the handler, and verifies the returned content.

- [ ] **Step 3: Implement MCP server**

Create `mcp/server.go` — Registers 6 tools with the mcp-go SDK using stdio transport. The server holds a `*engine.Game` and `[]*engine.AIPlayer` as state. `new_game` creates a fresh game. After each human action, AI opponents auto-play their turns before returning results. Tool responses include dice values, available categories, and scorecard in human-readable text format.

- [ ] **Step 4: Run tests**

```bash
go test ./mcp/ -v
```

Expected: all PASS.

- [ ] **Step 5: Wire up `yatz mcp` command**

Add to `cmd/yatz/main.go`:

```go
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for LLM integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return mcp.Serve()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
```

- [ ] **Step 6: Manual test with Claude Code**

Add to `.claude/settings.local.json` or equivalent, verify tools appear.

- [ ] **Step 7: Commit**

```bash
git add mcp/ cmd/
git commit -m "feat(mcp): add MCP server with game tools for LLM integration"
```

---

## Phase 4: P2P Online Play

### Task 9: P2P protocol definition

**Files:**
- Create: `p2p/protocol.go`
- Create: `p2p/protocol_test.go`

- [ ] **Step 1: Write failing tests for message serialization**

Create `p2p/protocol_test.go`:

```go
package p2p

import (
	"bytes"
	"testing"
)

func TestMessage_RoundTrip_Handshake(t *testing.T) {
	msg := NewHandshakeMsg("Alice")
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("write error: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if got.Type != MsgHandshake {
		t.Errorf("expected type %s, got %s", MsgHandshake, got.Type)
	}
	hs, err := DecodeHandshake(got)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if hs.Name != "Alice" {
		t.Errorf("expected Alice, got %s", hs.Name)
	}
}

func TestMessage_RoundTrip_Action(t *testing.T) {
	msg := NewActionMsg(ActionPayload{Action: ActionRoll})
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatalf("write error: %v", err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if got.Type != MsgAction {
		t.Errorf("expected type %s, got %s", MsgAction, got.Type)
	}
}
```

- [ ] **Step 2: Implement protocol types and JSON codec**

Create `p2p/protocol.go`:

```go
package p2p

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	MsgHandshake   = "handshake"
	MsgGameStart   = "game_start"
	MsgTurnStart   = "turn_start"
	MsgAction      = "action"
	MsgStateUpdate = "state_update"
	MsgGameOver    = "game_over"
	MsgError       = "error"
)

const (
	ActionRoll  = "roll"
	ActionHold  = "hold"
	ActionScore = "score"
)

type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type HandshakePayload struct {
	Name string `json:"name"`
}

type ActionPayload struct {
	Action   string `json:"action"`
	Indices  []int  `json:"indices,omitempty"`
	Category string `json:"category,omitempty"`
}

// WriteMessage writes a length-prefixed JSON message.
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ReadMessage reads a length-prefixed JSON message.
func ReadMessage(r io.Reader) (*Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func NewHandshakeMsg(name string) *Message {
	payload, _ := json.Marshal(HandshakePayload{Name: name})
	return &Message{Type: MsgHandshake, Payload: payload}
}

func NewActionMsg(ap ActionPayload) *Message {
	payload, _ := json.Marshal(ap)
	return &Message{Type: MsgAction, Payload: payload}
}

func DecodeHandshake(msg *Message) (*HandshakePayload, error) {
	if msg.Type != MsgHandshake {
		return nil, fmt.Errorf("expected handshake, got %s", msg.Type)
	}
	var hs HandshakePayload
	if err := json.Unmarshal(msg.Payload, &hs); err != nil {
		return nil, err
	}
	return &hs, nil
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./p2p/ -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add p2p/
git commit -m "feat(p2p): add protocol message types and JSON codec"
```

---

### Task 10: P2P host

**Files:**
- Create: `p2p/host.go`
- Create: `p2p/host_test.go`

- [ ] **Step 1: Write failing tests for host**

Create `p2p/host_test.go` — Integration test using loopback TCP. Start host on a random port, connect a mock client, exchange handshake, verify game_start is received, send actions, verify state_update responses.

- [ ] **Step 2: Implement host**

Create `p2p/host.go` — `Host` struct holds `engine.Game` and `engine.LocalClient`. Listens on TCP, accepts one connection, runs handshake, then game loop. On host's turn, reads from CLI input (via `GameClient` interface); on guest's turn, reads `action` messages from TCP and applies to engine. Broadcasts `state_update` and `turn_start` after each action.

- [ ] **Step 3: Run tests**

```bash
go test ./p2p/ -run TestHost -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add p2p/host.go p2p/host_test.go
git commit -m "feat(p2p): add host with game loop and state broadcasting"
```

---

### Task 11: P2P guest (RemoteClient)

**Files:**
- Create: `p2p/guest.go`
- Create: `p2p/guest_test.go`

- [ ] **Step 1: Write failing tests for guest/RemoteClient**

Create `p2p/guest_test.go` — Integration test: start a host on loopback, create guest, verify `RemoteClient` implements `GameClient` interface. Test Roll/Hold/Score through the TCP connection.

- [ ] **Step 2: Implement guest and RemoteClient**

Create `p2p/guest.go` — `Guest` connects to host, exchanges handshake. `RemoteClient` implements `GameClient` by sending `action` messages and waiting for `state_update` responses. Also listens for `turn_start`/`game_over` messages asynchronously.

- [ ] **Step 3: Run tests**

```bash
go test ./p2p/ -run TestGuest -v
```

Expected: all PASS.

- [ ] **Step 4: Wire up `yatz host` and `yatz join` commands**

Add to `cmd/yatz/main.go`:

```go
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host a P2P game",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		name, _ := cmd.Flags().GetString("name")
		return p2p.RunHost(port, name)
	},
}

var joinCmd = &cobra.Command{
	Use:   "join [address]",
	Short: "Join a P2P game",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		return p2p.RunGuest(args[0], name)
	},
}
```

- [ ] **Step 5: Manual test with two terminals**

```bash
# Terminal 1
./yatz host --port 9876 --name Alice

# Terminal 2
./yatz join localhost:9876 --name Bob
```

- [ ] **Step 6: Commit**

```bash
git add p2p/ cmd/
git commit -m "feat(p2p): add guest/RemoteClient and host/join commands"
```

---

## Phase 5: Matchmaking

### Task 12: Lambda matchmaking handler

**Files:**
- Create: `lambda/handler.go`
- Create: `lambda/handler_test.go`

- [ ] **Step 1: Install AWS dependencies**

```bash
go get github.com/aws/aws-lambda-go
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/config
```

- [ ] **Step 2: Write failing tests for Lambda handler**

Create `lambda/handler_test.go` — Test the matchmaking logic with a mock DynamoDB client. Test cases: first player registers and waits, second player triggers match and both get notified.

- [ ] **Step 3: Implement Lambda handler**

Create `lambda/handler.go` — Handles API Gateway WebSocket events ($connect, $disconnect, message). On connect: register in DynamoDB. On message (with port info): check for waiting players. If match found, notify both via API Gateway Management API with each other's endpoint.

- [ ] **Step 4: Run tests**

```bash
go test ./lambda/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add lambda/
git commit -m "feat(lambda): add serverless matchmaking handler"
```

---

### Task 13: Match client and `yatz match` command

**Files:**
- Create: `match/client.go`
- Create: `match/client_test.go`
- Modify: `cmd/yatz/main.go`

- [ ] **Step 1: Write failing tests for match client**

Create `match/client_test.go`:

```go
package match

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestMatchClient_ReceivesMatchResult(t *testing.T) {
	// Mock WebSocket server
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		// Read registration
		_, msg, _ := conn.ReadMessage()
		if !strings.Contains(string(msg), "port") {
			t.Error("expected port in registration")
		}
		// Send match result
		conn.WriteJSON(MatchResult{
			OpponentAddr: "192.168.1.10:9876",
			IsHost:       true,
		})
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	result, err := FindMatch(wsURL, "Alice", 9876)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsHost {
		t.Error("expected to be host")
	}
	if result.OpponentAddr != "192.168.1.10:9876" {
		t.Errorf("expected opponent addr, got %s", result.OpponentAddr)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./match/ -v
```

Expected: FAIL — `FindMatch` not defined.

- [ ] **Step 3: Implement matchmaking client**

Create `match/client.go` — `FindMatch(wsURL string, name string, port int) (*MatchResult, error)` connects to matchmaking WebSocket API, sends player info (name, listening port), waits for match notification. Returns the opponent's address and whether this client is host or guest.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./match/ -v
```

Expected: all PASS.

- [ ] **Step 5: Wire up `yatz match` command**

Add to `cmd/yatz/main.go` — The `match` command starts a TCP listener on a random port, connects to matchmaking API, waits for match. Once matched, either runs as host or connects as guest to the opponent.

- [ ] **Step 6: Commit**

```bash
git add match/ cmd/
git commit -m "feat(match): add matchmaking client and match command"
```

---

## Phase 6: Distribution and Polish

### Task 14: CI workflow + GoReleaser + GitHub Actions

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.goreleaser.yml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 0: Create CI workflow for PRs**

Create `.github/workflows/ci.yml`:

```yaml
name: CI
on:
  push:
    branches: [main, feature/**]
  pull_request:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./... -v -count=1
      - run: go build ./cmd/yatz/
```

- [ ] **Step 1: Create GoReleaser config**

Create `.goreleaser.yml`:

```yaml
version: 2
builds:
  - main: ./cmd/yatz
    binary: yatz
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
archives:
  - formats: ['tar.gz']
    format_overrides:
      - goos: windows
        formats: ['zip']
```

- [ ] **Step 2: Create GitHub Actions release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags:
      - 'v*'
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yml .github/
git commit -m "ci: add GoReleaser config and release workflow"
```

---

### Task 15: CLAUDE.md and README update

**Files:**
- Create: `CLAUDE.md`
- Modify: `README.md`

- [ ] **Step 1: Create project CLAUDE.md**

Create `CLAUDE.md` with project-specific instructions: build commands, test commands, project structure overview, key design decisions.

- [ ] **Step 2: Update README.md**

Update with new installation instructions (`go install`, binary download), usage for all subcommands, MCP configuration example.

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: update README and add project CLAUDE.md for v2"
```

---

### Task 16: Run full test suite and verify build

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v -count=1
```

Expected: all PASS.

- [ ] **Step 2: Build and verify binary**

```bash
go build -o yatz ./cmd/yatz/
./yatz --help
./yatz play --help
./yatz mcp --help
./yatz host --help
./yatz join --help
./yatz match --help
```

Expected: all subcommands show help text.

- [ ] **Step 3: Verify go install works**

```bash
go install ./cmd/yatz/
```

Expected: installs successfully.

- [ ] **Step 4: Final commit if any cleanup needed**

```bash
git add -A
git commit -m "chore: final cleanup for v2 release"
```
