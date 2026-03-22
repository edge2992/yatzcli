package engine

import "errors"

// HoldStep records a hold decision during a turn.
type HoldStep struct {
	Dice [5]int
	Held []int
}

type AIPlayer struct {
	game     *Game
	playerID string
	strategy Strategy
}

func NewAIPlayer(game *Game, playerID string) *AIPlayer {
	return &AIPlayer{game: game, playerID: playerID, strategy: &GreedyStrategy{}}
}

func NewAIPlayerWithStrategy(game *Game, playerID string, strategy Strategy) *AIPlayer {
	return &AIPlayer{game: game, playerID: playerID, strategy: strategy}
}

func (ai *AIPlayer) PlayTurn() (AITurnResult, error) {
	if ai.game.Players[ai.game.Current].ID != ai.playerID {
		return AITurnResult{}, errors.New("not AI's turn")
	}
	if err := ai.game.Roll(); err != nil {
		return AITurnResult{}, err
	}

	playerName := ai.game.Players[ai.game.Current].Name
	scorecard := ai.game.Players[ai.game.Current].Scorecard
	var holdHistory []HoldStep

	for {
		available := scorecard.AvailableCategories()
		action := ai.strategy.DecideAction(ai.game.Dice, ai.game.RollCount, scorecard, available)

		if action.Type == "hold" && ai.game.RollCount < MaxRolls {
			holdHistory = append(holdHistory, HoldStep{
				Dice: ai.game.Dice,
				Held: action.Indices,
			})
			if err := ai.game.Hold(action.Indices); err != nil {
				return AITurnResult{}, err
			}
			continue
		}

		// Score
		dice := ai.game.Dice
		category := action.Category
		score := CalcScore(category, dice)
		if err := ai.game.Score(category); err != nil {
			return AITurnResult{}, err
		}
		return AITurnResult{
			PlayerName:   playerName,
			Dice:         dice,
			Category:     category,
			Score:        score,
			StrategyName: ai.strategy.Name(),
			HoldHistory:  holdHistory,
		}, nil
	}
}
