package game

type ScoreCalculator func([]int) int

var scoreCalculators = map[ScoreCategory]ScoreCalculator{
	Ones:          calculateUpperSection(1),
	Twos:          calculateUpperSection(2),
	Threes:        calculateUpperSection(3),
	Fours:         calculateUpperSection(4),
	Fives:         calculateUpperSection(5),
	Sixes:         calculateUpperSection(6),
	ThreeOfAKind:  calculateThreeOfAKind,
	FourOfAKind:   calculateFourOfAKind,
	FullHouse:     calculateFullHouse,
	SmallStraight: calculateSmallStraight,
	LargeStraight: calculateLargeStraight,
	Yahtzee:       calculateYahtzee,
	Chance:        calculateChance,
}

func calculateUpperSection(number int) ScoreCalculator {
	return func(diceCounts []int) int {
		return diceCounts[number-1] * number
	}
}

func calculateThreeOfAKind(diceCounts []int) int {
	for _, count := range diceCounts {
		if count >= 3 {
			// Return sum of all dice
			total := 0
			for i, c := range diceCounts {
				total += (i + 1) * c
			}
			return total
		}
	}
	return 0
}

func calculateFourOfAKind(diceCounts []int) int {
	for _, count := range diceCounts {
		if count >= 4 {
			// Return sum of all dice
			total := 0
			for i, c := range diceCounts {
				total += (i + 1) * c
			}
			return total
		}
	}
	return 0
}

func calculateFullHouse(diceCounts []int) int {
	hasThree := false
	hasTwo := false
	for _, count := range diceCounts {
		if count == 2 {
			hasTwo = true
		} else if count == 3 {
			hasThree = true
		}
	}
	if hasTwo && hasThree {
		return 25
	}
	return 0
}

func calculateSmallStraight(diceCounts []int) int {
	if (diceCounts[0] > 0 && diceCounts[1] > 0 && diceCounts[2] > 0 && diceCounts[3] > 0) ||
		(diceCounts[1] > 0 && diceCounts[2] > 0 && diceCounts[3] > 0 && diceCounts[4] > 0) ||
		(diceCounts[2] > 0 && diceCounts[3] > 0 && diceCounts[4] > 0 && diceCounts[5] > 0) {
		return 30
	}
	return 0
}

func calculateLargeStraight(diceCounts []int) int {
	if (diceCounts[0] == 1 && diceCounts[1] == 1 && diceCounts[2] == 1 && diceCounts[3] == 1 && diceCounts[4] == 1) ||
		(diceCounts[1] == 1 && diceCounts[2] == 1 && diceCounts[3] == 1 && diceCounts[4] == 1 && diceCounts[5] == 1) {
		return 40
	}
	return 0
}

func calculateYahtzee(diceCounts []int) int {
	for _, count := range diceCounts {
		if count == 5 {
			return 50
		}
	}
	return 0
}

func calculateChance(diceCounts []int) int {
	total := 0
	for i, count := range diceCounts {
		total += (i + 1) * count
	}
	return total
}

func CalculateScore(dice []Dice, category ScoreCategory) int {
	diceCounts := countDiceValues(dice)
	calculator, ok := scoreCalculators[category]

	if !ok {
		return 0
	}

	return calculator(diceCounts)
}

func countDiceValues(dice []Dice) []int {
	counts := make([]int, 6)
	for _, die := range dice {
		counts[die.Value-1]++
	}
	return counts
}

func CalculateTotalScore(scoreCard ScoreCard) int {
	total := 0

	// Calculate Upper Section total
	upperSectionTotal := 0
	upperSectionCategories := []ScoreCategory{Ones, Twos, Threes, Fours, Fives, Sixes}
	for _, category := range upperSectionCategories {
		upperSectionTotal += scoreCard.Scores[category]
	}

	// Add Upper Section bonus if >= 63
	const upperSectionBonusThreshold = 63
	const upperSectionBonus = 35
	total += upperSectionTotal
	if upperSectionTotal >= upperSectionBonusThreshold {
		total += upperSectionBonus
	}

	// Add Lower Section scores
	lowerSectionCategories := []ScoreCategory{
		ThreeOfAKind, FourOfAKind, FullHouse,
		SmallStraight, LargeStraight, Yahtzee, Chance,
	}
	for _, category := range lowerSectionCategories {
		total += scoreCard.Scores[category]
	}

	return total
}
