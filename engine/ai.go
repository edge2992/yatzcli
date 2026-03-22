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
	best := ai.bestCategory()
	return ai.game.Score(best)
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
