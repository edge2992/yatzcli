package engine

import "sort"

func CalcScore(c Category, dice [5]int) int {
	switch c {
	case Ones:
		return countValue(dice, 1)
	case Twos:
		return countValue(dice, 2) * 2
	case Threes:
		return countValue(dice, 3) * 3
	case Fours:
		return countValue(dice, 4) * 4
	case Fives:
		return countValue(dice, 5) * 5
	case Sixes:
		return countValue(dice, 6) * 6
	case ThreeOfAKind:
		if hasNOfAKind(dice, 3) {
			return sum(dice)
		}
		return 0
	case FourOfAKind:
		if hasNOfAKind(dice, 4) {
			return sum(dice)
		}
		return 0
	case FullHouse:
		if isFullHouse(dice) {
			return 25
		}
		return 0
	case SmallStraight:
		if hasStraight(dice, 4) {
			return 30
		}
		return 0
	case LargeStraight:
		if hasStraight(dice, 5) {
			return 40
		}
		return 0
	case Yahtzee:
		if hasNOfAKind(dice, 5) {
			return 50
		}
		return 0
	case Chance:
		return sum(dice)
	}
	return 0
}

func countValue(dice [5]int, val int) int {
	n := 0
	for _, d := range dice {
		if d == val {
			n++
		}
	}
	return n
}

func sum(dice [5]int) int {
	total := 0
	for _, d := range dice {
		total += d
	}
	return total
}

func counts(dice [5]int) map[int]int {
	m := make(map[int]int)
	for _, d := range dice {
		m[d]++
	}
	return m
}

func hasNOfAKind(dice [5]int, n int) bool {
	for _, cnt := range counts(dice) {
		if cnt >= n {
			return true
		}
	}
	return false
}

func isFullHouse(dice [5]int) bool {
	c := counts(dice)
	if len(c) != 2 {
		return false
	}
	for _, cnt := range c {
		if cnt == 2 || cnt == 3 {
			return true
		}
	}
	return false
}

func hasStraight(dice [5]int, length int) bool {
	sorted := make([]int, len(dice))
	copy(sorted, dice[:])
	sort.Ints(sorted)

	// deduplicate
	uniq := []int{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			uniq = append(uniq, sorted[i])
		}
	}

	if len(uniq) < length {
		return false
	}

	run := 1
	for i := 1; i < len(uniq); i++ {
		if uniq[i] == uniq[i-1]+1 {
			run++
			if run >= length {
				return true
			}
		} else {
			run = 1
		}
	}
	return false
}
