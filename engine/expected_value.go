package engine

// allHoldCombinations contains all 32 possible hold combinations (subsets of {0,1,2,3,4}).
// Computed once at package init time.
var allHoldCombinations = func() [][]int {
	combos := make([][]int, 0, 32)
	for mask := 0; mask < 32; mask++ {
		var indices []int
		for bit := 0; bit < 5; bit++ {
			if mask&(1<<bit) != 0 {
				indices = append(indices, bit)
			}
		}
		combos = append(combos, indices)
	}
	return combos
}()

func holdCombinations() [][]int {
	return allHoldCombinations
}

// expectedValue calculates the average best score across all possible reroll outcomes
// for a given hold combination.
func expectedValue(dice [5]int, hold []int, available []Category, scorecard Scorecard) float64 {
	holdSet := make(map[int]bool, len(hold))
	for _, i := range hold {
		holdSet[i] = true
	}

	// Count free dice (not held)
	var freeDice int
	for i := 0; i < 5; i++ {
		if !holdSet[i] {
			freeDice++
		}
	}

	if freeDice == 0 {
		return float64(bestScoreForDice(dice, available))
	}

	totalOutcomes := pow6(freeDice)
	totalScore := 0.0

	// Enumerate all outcomes for free dice
	for outcome := 0; outcome < totalOutcomes; outcome++ {
		var newDice [5]int
		copy(newDice[:], dice[:])
		rem := outcome
		for i := 0; i < 5; i++ {
			if !holdSet[i] {
				newDice[i] = (rem % 6) + 1
				rem /= 6
			}
		}
		best := bestScoreForDice(newDice, available)
		totalScore += float64(best)
	}

	return totalScore / float64(totalOutcomes)
}

// expectedValueWithBonus adds upper bonus consideration to the expected value.
func expectedValueWithBonus(dice [5]int, hold []int, available []Category, scorecard Scorecard) float64 {
	base := expectedValue(dice, hold, available, scorecard)

	// Bonus: if close to upper bonus, add incentive for upper section scores
	upperTotal := scorecard.UpperTotal()
	if upperTotal < UpperBonusThreshold && !scorecard.HasUpperBonus() {
		remaining := UpperBonusThreshold - upperTotal
		// Count remaining upper categories
		upperRemaining := 0
		for _, c := range UpperCategories {
			if !scorecard.IsFilled(c) {
				upperRemaining++
			}
		}
		if upperRemaining > 0 && remaining <= upperRemaining*5 {
			// Close to bonus — small boost
			base += float64(UpperBonusValue) * 0.1
		}
	}

	return base
}

func bestScoreForDice(dice [5]int, available []Category) int {
	best := 0
	for _, c := range available {
		s := CalcScore(c, dice)
		if s > best {
			best = s
		}
	}
	return best
}

func pow6(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 6
	}
	return result
}
