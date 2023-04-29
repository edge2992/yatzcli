package game

type PlayerInfo struct {
	Name      string
	ScoreCard ScoreCard
}

func CreateDices() []Dice {
	dices := make([]Dice, NumberOfDice)
	return dices
}
