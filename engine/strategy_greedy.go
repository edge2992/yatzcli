package engine

// GreedyStrategy always scores immediately with the highest-scoring available category.
// It never uses Hold to reroll.
type GreedyStrategy struct{}

func (s *GreedyStrategy) Name() string { return "greedy" }

func (s *GreedyStrategy) DecideAction(dice [5]int, rollCount int, scorecard Scorecard, available []Category) TurnAction {
	return TurnAction{Type: "score", Category: bestCategoryForDice(dice, available)}
}
