package engine

import "math/rand"

func RollAll(src rand.Source) [5]int {
	r := rand.New(src)
	var dice [5]int
	for i := range dice {
		dice[i] = r.Intn(6) + 1
	}
	return dice
}

func Reroll(dice [5]int, hold []int, src rand.Source) [5]int {
	holdSet := make(map[int]bool)
	for _, i := range hold {
		holdSet[i] = true
	}
	r := rand.New(src)
	var result [5]int
	for i := range dice {
		if holdSet[i] {
			result[i] = dice[i]
		} else {
			result[i] = r.Intn(6) + 1
		}
	}
	return result
}
