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
	for i, count := range diceCounts {
		if count >= 3 {
			return (i + 1) * 3
		}
	}
	return 0
}

func calculateFourOfAKind(diceCounts []int) int {
	for i, count := range diceCounts {
		if count >= 4 {
			return (i + 1) * 4
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

func calculateTotalScore(scoreCard ScoreCard) int {
	total := 0
	for _, score := range scoreCard.Scores {
		total += score
	}
	return total
}
