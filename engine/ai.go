package engine

import "errors"

type AIPlayer struct {
	game     *Game
	playerID string
}

func NewAIPlayer(game *Game, playerID string) *AIPlayer {
	return &AIPlayer{game: game, playerID: playerID}
}

func (ai *AIPlayer) PlayTurn() (AITurnResult, error) {
	if ai.game.Players[ai.game.Current].ID != ai.playerID {
		return AITurnResult{}, errors.New("not AI's turn")
	}
	if err := ai.game.Roll(); err != nil {
		return AITurnResult{}, err
	}
	dice := ai.game.Dice
	best := ai.bestCategory()
	score := CalcScore(best, dice)
	playerName := ai.game.Players[ai.game.Current].Name
	if err := ai.game.Score(best); err != nil {
		return AITurnResult{}, err
	}
	return AITurnResult{
		PlayerName: playerName,
		Dice:       dice,
		Category:   best,
		Score:      score,
	}, nil
}

func (ai *AIPlayer) bestCategory() Category {
	avail := ai.game.Players[ai.game.Current].Scorecard.AvailableCategories()
	bestCat := avail[0]
	bestScore := CalcScore(bestCat, ai.game.Dice)
	for _, c := range avail[1:] {
		s := CalcScore(c, ai.game.Dice)
		if s > bestScore {
			bestScore = s
			bestCat = c
		}
	}
	return bestCat
}
