package engine

import (
	"math/rand"
	"testing"
)

func TestRollAll(t *testing.T) {
	src := rand.NewSource(42)
	dice := RollAll(src)
	for i, d := range dice {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, d)
		}
	}
}

func TestRollAll_Deterministic(t *testing.T) {
	d1 := RollAll(rand.NewSource(42))
	d2 := RollAll(rand.NewSource(42))
	if d1 != d2 {
		t.Errorf("same seed should produce same dice: %v != %v", d1, d2)
	}
}

func TestReroll(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	hold := []int{0, 2, 4}
	result := Reroll(original, hold, src)
	if result[0] != 1 || result[2] != 3 || result[4] != 5 {
		t.Errorf("held dice changed: %v", result)
	}
	for _, i := range []int{1, 3} {
		if result[i] < 1 || result[i] > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, result[i])
		}
	}
}

func TestReroll_HoldAll(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	hold := []int{0, 1, 2, 3, 4}
	result := Reroll(original, hold, src)
	if result != original {
		t.Errorf("holding all should not change dice: %v != %v", result, original)
	}
}

func TestReroll_HoldNone(t *testing.T) {
	src := rand.NewSource(42)
	original := [5]int{1, 2, 3, 4, 5}
	result := Reroll(original, nil, src)
	for i, d := range result {
		if d < 1 || d > 6 {
			t.Errorf("dice[%d] = %d, out of range", i, d)
		}
	}
}
