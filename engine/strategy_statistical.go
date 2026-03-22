package engine

// StatisticalStrategy uses expected value calculation to decide whether to hold or score.
// On the 3rd roll it always scores the best category.
// Otherwise it compares immediate scoring vs expected value of each hold combination.
type StatisticalStrategy struct{}

func (s *StatisticalStrategy) Name() string { return "statistical" }

func (s *StatisticalStrategy) DecideAction(dice [5]int, rollCount int, scorecard Scorecard, available []Category) TurnAction {
	// 3rd roll: must score
	if rollCount >= MaxRolls {
		return TurnAction{Type: "score", Category: bestCategoryForDice(dice, available)}
	}

	// Immediate best score
	immediateBest := bestCategoryForDice(dice, available)
	immediateScore := float64(CalcScore(immediateBest, dice))

	// Find the best hold combination by expected value
	bestEV := immediateScore
	var bestHold []int

	for _, hold := range holdCombinations() {
		if len(hold) == 5 {
			// Holding all dice is the same as immediate scoring
			continue
		}
		ev := expectedValueWithBonus(dice, hold, available, scorecard)
		if ev > bestEV {
			bestEV = ev
			bestHold = hold
		}
	}

	if bestHold != nil {
		return TurnAction{Type: "hold", Indices: bestHold}
	}

	return TurnAction{Type: "score", Category: immediateBest}
}

// bestCategoryForDice returns the highest-scoring available category.
// Panics if available is empty; callers must ensure at least one category.
func bestCategoryForDice(dice [5]int, available []Category) Category {
	if len(available) == 0 {
		return Chance
	}
	bestCat := available[0]
	bestScore := CalcScore(bestCat, dice)
	for _, c := range available[1:] {
		s := CalcScore(c, dice)
		if s > bestScore {
			bestScore = s
			bestCat = c
		}
	}
	return bestCat
}
