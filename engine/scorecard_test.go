package engine

import (
	"encoding/json"
	"testing"
)

func TestNewScorecard(t *testing.T) {
	sc := NewScorecard()
	for _, c := range AllCategories {
		if sc.IsFilled(c) {
			t.Errorf("expected category %s to be unfilled", c)
		}
	}
}

func TestFill(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)

	if !sc.IsFilled(Ones) {
		t.Error("expected Ones to be filled")
	}
	if got := sc.GetScore(Ones); got != 3 {
		t.Errorf("expected score 3, got %d", got)
	}
}

func TestFillZero(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Yahtzee, 0)

	if !sc.IsFilled(Yahtzee) {
		t.Error("expected Yahtzee to be filled even with score 0")
	}
	if got := sc.GetScore(Yahtzee); got != 0 {
		t.Errorf("expected score 0, got %d", got)
	}
}

func TestAvailableCategories(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)
	sc.Fill(Twos, 6)

	avail := sc.AvailableCategories()
	for _, c := range avail {
		if c == Ones || c == Twos {
			t.Errorf("expected %s to be excluded from available categories", c)
		}
	}

	expectedLen := len(AllCategories) - 2
	if len(avail) != expectedLen {
		t.Errorf("expected %d available categories, got %d", expectedLen, len(avail))
	}
}

func TestUpperBonus(t *testing.T) {
	sc := NewScorecard()
	// Fill upper section to exactly reach threshold: 3+6+9+12+15+18 = 63
	sc.Fill(Ones, 3)
	sc.Fill(Twos, 6)
	sc.Fill(Threes, 9)
	sc.Fill(Fours, 12)
	sc.Fill(Fives, 15)
	sc.Fill(Sixes, 18)

	if !sc.HasUpperBonus() {
		t.Error("expected upper bonus when upper total is 63")
	}
	if got := sc.UpperTotal(); got != 63 {
		t.Errorf("expected upper total 63, got %d", got)
	}
}

func TestNoUpperBonus(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 2)
	sc.Fill(Twos, 4)
	sc.Fill(Threes, 6)
	sc.Fill(Fours, 8)
	sc.Fill(Fives, 10)
	sc.Fill(Sixes, 12)

	if sc.HasUpperBonus() {
		t.Error("expected no upper bonus when upper total is 42")
	}
}

func TestTotal(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)
	sc.Fill(Twos, 6)
	sc.Fill(Threes, 9)
	sc.Fill(Fours, 12)
	sc.Fill(Fives, 15)
	sc.Fill(Sixes, 18)
	sc.Fill(Chance, 20)

	// Upper total = 63, has bonus = 35, chance = 20 => total = 63 + 35 + 20 = 118
	if got := sc.Total(); got != 118 {
		t.Errorf("expected total 118, got %d", got)
	}
}

func TestScorecardJSON(t *testing.T) {
	sc := NewScorecard()
	sc.Fill(Ones, 3)
	sc.Fill(Yahtzee, 0)

	data, err := json.Marshal(sc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var sc2 Scorecard
	if err := json.Unmarshal(data, &sc2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if !sc2.IsFilled(Ones) {
		t.Error("expected Ones to be filled after round-trip")
	}
	if got := sc2.GetScore(Ones); got != 3 {
		t.Errorf("expected Ones score 3, got %d", got)
	}
	if !sc2.IsFilled(Yahtzee) {
		t.Error("expected Yahtzee to be filled (zero score) after round-trip")
	}
	if got := sc2.GetScore(Yahtzee); got != 0 {
		t.Errorf("expected Yahtzee score 0, got %d", got)
	}
	if sc2.IsFilled(Twos) {
		t.Error("expected Twos to be unfilled after round-trip")
	}
}
