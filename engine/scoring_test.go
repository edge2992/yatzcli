package engine

import "testing"

func TestUpperSection(t *testing.T) {
	dice := [5]int{1, 2, 3, 4, 5}

	tests := []struct {
		cat  Category
		want int
	}{
		{Ones, 1},
		{Twos, 2},
		{Threes, 3},
		{Fours, 4},
		{Fives, 5},
		{Sixes, 0},
	}
	for _, tt := range tests {
		if got := CalcScore(tt.cat, dice); got != tt.want {
			t.Errorf("CalcScore(%s, %v) = %d, want %d", tt.cat, dice, got, tt.want)
		}
	}
}

func TestUpperSectionMultiple(t *testing.T) {
	dice := [5]int{3, 3, 3, 2, 1}
	if got := CalcScore(Threes, dice); got != 9 {
		t.Errorf("CalcScore(Threes, %v) = %d, want 9", dice, got)
	}
}

func TestThreeOfAKindValid(t *testing.T) {
	dice := [5]int{3, 3, 3, 4, 5}
	if got := CalcScore(ThreeOfAKind, dice); got != 18 {
		t.Errorf("CalcScore(ThreeOfAKind, %v) = %d, want 18", dice, got)
	}
}

func TestThreeOfAKindInvalid(t *testing.T) {
	dice := [5]int{1, 2, 3, 4, 5}
	if got := CalcScore(ThreeOfAKind, dice); got != 0 {
		t.Errorf("CalcScore(ThreeOfAKind, %v) = %d, want 0", dice, got)
	}
}

func TestFourOfAKindValid(t *testing.T) {
	dice := [5]int{4, 4, 4, 4, 2}
	if got := CalcScore(FourOfAKind, dice); got != 18 {
		t.Errorf("CalcScore(FourOfAKind, %v) = %d, want 18", dice, got)
	}
}

func TestFourOfAKindInvalid(t *testing.T) {
	dice := [5]int{3, 3, 3, 4, 5}
	if got := CalcScore(FourOfAKind, dice); got != 0 {
		t.Errorf("CalcScore(FourOfAKind, %v) = %d, want 0", dice, got)
	}
}

func TestFullHouseValid(t *testing.T) {
	dice := [5]int{2, 2, 3, 3, 3}
	if got := CalcScore(FullHouse, dice); got != 25 {
		t.Errorf("CalcScore(FullHouse, %v) = %d, want 25", dice, got)
	}
}

func TestFullHouseInvalid(t *testing.T) {
	dice := [5]int{1, 2, 3, 4, 5}
	if got := CalcScore(FullHouse, dice); got != 0 {
		t.Errorf("CalcScore(FullHouse, %v) = %d, want 0", dice, got)
	}
}

func TestFullHouseYahtzeeIsNotFullHouse(t *testing.T) {
	dice := [5]int{4, 4, 4, 4, 4}
	if got := CalcScore(FullHouse, dice); got != 0 {
		t.Errorf("CalcScore(FullHouse, %v) = %d, want 0 (Yahtzee is not FullHouse)", dice, got)
	}
}

func TestSmallStraightValid(t *testing.T) {
	tests := []struct {
		dice [5]int
	}{
		{[5]int{1, 2, 3, 4, 6}},
		{[5]int{2, 3, 4, 5, 1}},
		{[5]int{3, 4, 5, 6, 1}},
		{[5]int{1, 2, 3, 4, 4}},
	}
	for _, tt := range tests {
		if got := CalcScore(SmallStraight, tt.dice); got != 30 {
			t.Errorf("CalcScore(SmallStraight, %v) = %d, want 30", tt.dice, got)
		}
	}
}

func TestSmallStraightInvalid(t *testing.T) {
	dice := [5]int{1, 2, 4, 5, 6}
	if got := CalcScore(SmallStraight, dice); got != 0 {
		t.Errorf("CalcScore(SmallStraight, %v) = %d, want 0", dice, got)
	}
}

func TestLargeStraightValid(t *testing.T) {
	tests := []struct {
		dice [5]int
	}{
		{[5]int{1, 2, 3, 4, 5}},
		{[5]int{2, 3, 4, 5, 6}},
	}
	for _, tt := range tests {
		if got := CalcScore(LargeStraight, tt.dice); got != 40 {
			t.Errorf("CalcScore(LargeStraight, %v) = %d, want 40", tt.dice, got)
		}
	}
}

func TestLargeStraightInvalid(t *testing.T) {
	dice := [5]int{1, 2, 3, 4, 6}
	if got := CalcScore(LargeStraight, dice); got != 0 {
		t.Errorf("CalcScore(LargeStraight, %v) = %d, want 0", dice, got)
	}
}

func TestYahtzeeValid(t *testing.T) {
	dice := [5]int{5, 5, 5, 5, 5}
	if got := CalcScore(Yahtzee, dice); got != 50 {
		t.Errorf("CalcScore(Yahtzee, %v) = %d, want 50", dice, got)
	}
}

func TestYahtzeeInvalid(t *testing.T) {
	dice := [5]int{5, 5, 5, 5, 4}
	if got := CalcScore(Yahtzee, dice); got != 0 {
		t.Errorf("CalcScore(Yahtzee, %v) = %d, want 0", dice, got)
	}
}

func TestChance(t *testing.T) {
	dice := [5]int{1, 2, 3, 4, 5}
	if got := CalcScore(Chance, dice); got != 15 {
		t.Errorf("CalcScore(Chance, %v) = %d, want 15", dice, got)
	}
}
