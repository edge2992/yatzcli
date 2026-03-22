package engine

import "encoding/json"

type Scorecard struct {
	scores map[Category]*int
}

func (sc Scorecard) MarshalJSON() ([]byte, error) {
	m := make(map[string]*int, len(sc.scores))
	for k, v := range sc.scores {
		m[string(k)] = v
	}
	return json.Marshal(m)
}

func (sc *Scorecard) UnmarshalJSON(data []byte) error {
	var m map[string]*int
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	sc.scores = make(map[Category]*int, len(m))
	for k, v := range m {
		sc.scores[Category(k)] = v
	}
	return nil
}

func NewScorecard() Scorecard {
	return Scorecard{scores: make(map[Category]*int)}
}

func (sc *Scorecard) Fill(c Category, score int) {
	s := score
	sc.scores[c] = &s
}

func (sc *Scorecard) IsFilled(c Category) bool {
	return sc.scores[c] != nil
}

func (sc *Scorecard) GetScore(c Category) int {
	if sc.scores[c] == nil {
		return 0
	}
	return *sc.scores[c]
}

func (sc *Scorecard) AvailableCategories() []Category {
	var avail []Category
	for _, c := range AllCategories {
		if !sc.IsFilled(c) {
			avail = append(avail, c)
		}
	}
	return avail
}

func (sc *Scorecard) UpperTotal() int {
	total := 0
	for _, c := range UpperCategories {
		total += sc.GetScore(c)
	}
	return total
}

func (sc *Scorecard) HasUpperBonus() bool {
	return sc.UpperTotal() >= UpperBonusThreshold
}

func (sc *Scorecard) Total() int {
	total := 0
	for _, c := range AllCategories {
		total += sc.GetScore(c)
	}
	if sc.HasUpperBonus() {
		total += UpperBonusValue
	}
	return total
}
