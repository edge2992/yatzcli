package engine

// TurnAction represents a decision made by a Strategy during a turn.
type TurnAction struct {
	Type     string   // "hold" or "score"
	Indices  []int    // hold: dice indices to keep
	Category Category // score: category to score in
}

// Strategy defines the interface for AI decision-making.
type Strategy interface {
	Name() string
	DecideAction(dice [5]int, rollCount int, scorecard Scorecard, available []Category) TurnAction
}
