package engine

type Category string

const (
	Ones          Category = "ones"
	Twos          Category = "twos"
	Threes        Category = "threes"
	Fours         Category = "fours"
	Fives         Category = "fives"
	Sixes         Category = "sixes"
	ThreeOfAKind  Category = "three_of_a_kind"
	FourOfAKind   Category = "four_of_a_kind"
	FullHouse     Category = "full_house"
	SmallStraight Category = "small_straight"
	LargeStraight Category = "large_straight"
	Yahtzee       Category = "yahtzee"
	Chance        Category = "chance"
)

var AllCategories = []Category{
	Ones, Twos, Threes, Fours, Fives, Sixes,
	ThreeOfAKind, FourOfAKind, FullHouse,
	SmallStraight, LargeStraight, Yahtzee, Chance,
}

var UpperCategories = []Category{Ones, Twos, Threes, Fours, Fives, Sixes}

const UpperBonusThreshold = 63
const UpperBonusValue = 35
