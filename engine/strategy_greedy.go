package engine

// GreedyStrategy always scores immediately with the highest-scoring available category.
// It never uses Hold to reroll.
type GreedyStrategy struct{}

func (s *GreedyStrategy) Name() string { return "greedy" }

func (s *GreedyStrategy) DecideAction(dice [5]int, rollCount int, scorecard Scorecard, available []Category) TurnAction {
	bestCat := available[0]
	bestScore := CalcScore(bestCat, dice)
	for _, c := range available[1:] {
		s := CalcScore(c, dice)
		if s > bestScore {
			bestScore = s
			bestCat = c
		}
	}
	return TurnAction{Type: "score", Category: bestCat}
}
