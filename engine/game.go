package engine

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type GamePhase int

const (
	PhaseWaiting  GamePhase = iota // reserved, unused in v1
	PhaseRolling                   // Player can Roll() or Hold()
	PhaseChoosing                  // Player must Score() (after 3 rolls)
	PhaseFinished                  // Game over
)

const MaxRolls = 3
const MaxRounds = 13

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
	Players             []PlayerState
	CurrentPlayer       string
	CurrentPlayerIndex  int
	Round               int
	Dice                [5]int
	RollCount           int
	Phase               GamePhase
	AvailableCategories []Category
}

type PlayerState struct {
	ID        string
	Name      string
	Scorecard Scorecard
}

func NewGame(playerNames []string, src rand.Source) *Game {
	if len(playerNames) == 0 {
		panic("NewGame requires at least 1 player")
	}
	if src == nil {
		src = rand.NewSource(time.Now().UnixNano())
	}
	players := make([]Player, len(playerNames))
	for i, name := range playerNames {
		players[i] = Player{
			ID:        fmt.Sprintf("player-%d", i),
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
		return errors.New("cannot roll: not in rolling phase")
	}
	if g.RollCount != 0 {
		return errors.New("cannot roll: use Hold() for subsequent rolls")
	}
	g.Dice = RollAll(g.rng)
	g.RollCount++
	return nil
}

func (g *Game) Hold(indices []int) error {
	if g.Phase != PhaseRolling {
		return errors.New("cannot hold: not in rolling phase")
	}
	if g.RollCount == 0 {
		return errors.New("cannot hold: must Roll() first")
	}
	if g.RollCount >= MaxRolls {
		return errors.New("cannot hold: max rolls reached")
	}
	for _, idx := range indices {
		if idx < 0 || idx > 4 {
			return fmt.Errorf("cannot hold: index %d out of range (0-4)", idx)
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
	if g.Phase == PhaseFinished {
		return errors.New("cannot score: game is finished")
	}
	if g.RollCount == 0 {
		return errors.New("cannot score: must roll first")
	}
	if !IsValidCategory(category) {
		return fmt.Errorf("cannot score: invalid category %q", category)
	}
	player := &g.Players[g.Current]
	if player.Scorecard.IsFilled(category) {
		return fmt.Errorf("cannot score: category %s already filled", category)
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
		players[i] = PlayerState{
			ID:        p.ID,
			Name:      p.Name,
			Scorecard: p.Scorecard,
		}
	}
	return GameState{
		Players:             players,
		CurrentPlayer:       g.Players[g.Current].ID,
		CurrentPlayerIndex:  g.Current,
		Round:               g.Round,
		Dice:                g.Dice,
		RollCount:           g.RollCount,
		Phase:               g.Phase,
		AvailableCategories: g.GetAvailableCategories(),
	}
}

func (g *Game) GetAvailableCategories() []Category {
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
