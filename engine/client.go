package engine

type GameClient interface {
	Roll() (*GameState, error)
	Hold(indices []int) (*GameState, error)
	Score(category Category) (*GameState, error)
	GetState() (*GameState, error)
}

type AITurnResult struct {
	PlayerName   string
	Dice         [5]int
	Category     Category
	Score        int
	StrategyName string
	HoldHistory  []HoldStep
}

type LocalClient struct {
	game          *Game
	playerID      string
	ais           []*AIPlayer
	LastAIResults []AITurnResult
}

func NewLocalClient(game *Game, playerID string, ais []*AIPlayer) *LocalClient {
	return &LocalClient{game: game, playerID: playerID, ais: ais}
}

func (c *LocalClient) Roll() (*GameState, error) {
	if err := c.game.Roll(); err != nil {
		return nil, err
	}
	s := c.game.GetState()
	return &s, nil
}

func (c *LocalClient) Hold(indices []int) (*GameState, error) {
	if err := c.game.Hold(indices); err != nil {
		return nil, err
	}
	s := c.game.GetState()
	return &s, nil
}

func (c *LocalClient) Score(category Category) (*GameState, error) {
	if err := c.game.Score(category); err != nil {
		return nil, err
	}
	if err := c.runAITurns(); err != nil {
		return nil, err
	}
	s := c.game.GetState()
	return &s, nil
}

func (c *LocalClient) GetState() (*GameState, error) {
	s := c.game.GetState()
	return &s, nil
}

func (c *LocalClient) runAITurns() error {
	c.LastAIResults = nil
	for c.game.Phase != PhaseFinished {
		current := c.game.Players[c.game.Current]
		if current.ID == c.playerID {
			break
		}
		found := false
		for _, ai := range c.ais {
			if ai.playerID == current.ID {
				result, err := ai.PlayTurn()
				if err != nil {
					return err
				}
				c.LastAIResults = append(c.LastAIResults, result)
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return nil
}
