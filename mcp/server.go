package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/edge2992/yatzcli/engine"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type gameServer struct {
	game   *engine.Game
	client *engine.LocalClient
	ais    []*engine.AIPlayer
}

func Serve() error {
	s := newServer()
	return server.ServeStdio(s)
}

func newServer() *server.MCPServer {
	gs := &gameServer{}

	s := server.NewMCPServer(
		"yatzcli",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	newGameTool := mcp.NewTool("new_game",
		mcp.WithDescription("Start a new Yahtzee game with AI opponents"),
		mcp.WithNumber("opponents", mcp.Description("Number of AI opponents (1-3, default 1)")),
	)
	s.AddTool(newGameTool, gs.handleNewGame)

	rollDiceTool := mcp.NewTool("roll_dice",
		mcp.WithDescription("Roll all five dice (first roll of a turn)"),
	)
	s.AddTool(rollDiceTool, gs.handleRollDice)

	holdDiceTool := mcp.NewTool("hold_dice",
		mcp.WithDescription("Hold specified dice and reroll the others. Indices are 0-4."),
		mcp.WithArray("indices",
			mcp.Required(),
			mcp.Description("Array of dice indices to hold, e.g. [0,2,4]"),
			mcp.Items(map[string]any{"type": "integer"}),
		),
	)
	s.AddTool(holdDiceTool, gs.handleHoldDice)

	scoreTool := mcp.NewTool("score",
		mcp.WithDescription("Choose a scoring category for the current dice"),
		mcp.WithString("category", mcp.Required(), mcp.Description("Scoring category name (e.g. ones, twos, full_house, yahtzee)")),
	)
	s.AddTool(scoreTool, gs.handleScore)

	getStateTool := mcp.NewTool("get_state",
		mcp.WithDescription("Get the current game state"),
	)
	s.AddTool(getStateTool, gs.handleGetState)

	getScorecardTool := mcp.NewTool("get_scorecard",
		mcp.WithDescription("Get scorecard for a player or all players"),
		mcp.WithString("player_id", mcp.Description("Player ID (omit for all players)")),
	)
	s.AddTool(getScorecardTool, gs.handleGetScorecard)

	return s
}

func (gs *gameServer) handleNewGame(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opponents := req.GetInt("opponents", 1)
	if opponents < 1 {
		opponents = 1
	}
	if opponents > 3 {
		opponents = 3
	}

	names := []string{"You"}
	for i := 0; i < opponents; i++ {
		names = append(names, fmt.Sprintf("AI-%d", i+1))
	}

	gs.game = engine.NewGame(names, nil)
	gs.ais = make([]*engine.AIPlayer, opponents)
	for i := 0; i < opponents; i++ {
		gs.ais[i] = engine.NewAIPlayer(gs.game, fmt.Sprintf("player-%d", i+1))
	}
	gs.client = engine.NewLocalClient(gs.game, "player-0", gs.ais)

	state, _ := gs.client.GetState()
	return mcp.NewToolResultText(fmt.Sprintf(
		"New game started with %d AI opponent(s)!\n\n%s",
		opponents, formatState(state),
	)), nil
}

func (gs *gameServer) handleRollDice(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if gs.client == nil {
		return mcp.NewToolResultError("No game in progress. Use new_game first."), nil
	}
	state, err := gs.client.Roll()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(
		"Rolled!\n\n%s", formatDiceAndState(state),
	)), nil
}

func (gs *gameServer) handleHoldDice(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if gs.client == nil {
		return mcp.NewToolResultError("No game in progress. Use new_game first."), nil
	}
	indices := req.GetIntSlice("indices", nil)
	if indices == nil {
		return mcp.NewToolResultError("indices parameter is required (JSON array of ints 0-4)"), nil
	}
	state, err := gs.client.Hold(indices)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf(
		"Held dice at indices %v and rerolled others.\n\n%s", indices, formatDiceAndState(state),
	)), nil
}

func (gs *gameServer) handleScore(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if gs.client == nil {
		return mcp.NewToolResultError("No game in progress. Use new_game first."), nil
	}
	category, err := req.RequireString("category")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cat := engine.Category(category)
	score := engine.CalcScore(cat, gs.game.Dice)

	state, scoreErr := gs.client.Score(cat)
	if scoreErr != nil {
		return mcp.NewToolResultError(scoreErr.Error()), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Scored %d points in %s.\n\n", score, category)
	if state.Phase == engine.PhaseFinished {
		sb.WriteString("Game Over!\n\n")
		sb.WriteString(formatFinalScores(state))
	} else {
		sb.WriteString(formatState(state))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func (gs *gameServer) handleGetState(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if gs.client == nil {
		return mcp.NewToolResultError("No game in progress. Use new_game first."), nil
	}
	state, _ := gs.client.GetState()
	return mcp.NewToolResultText(formatState(state)), nil
}

func (gs *gameServer) handleGetScorecard(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if gs.client == nil {
		return mcp.NewToolResultError("No game in progress. Use new_game first."), nil
	}
	playerID := req.GetString("player_id", "")
	state, _ := gs.client.GetState()

	if playerID != "" {
		for _, p := range state.Players {
			if p.ID == playerID {
				return mcp.NewToolResultText(formatPlayerScorecard(p)), nil
			}
		}
		return mcp.NewToolResultError(fmt.Sprintf("Player %q not found.", playerID)), nil
	}

	var sb strings.Builder
	for i, p := range state.Players {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(formatPlayerScorecard(p))
	}
	return mcp.NewToolResultText(sb.String()), nil
}

func formatDice(dice [5]int) string {
	parts := make([]string, 5)
	for i, d := range dice {
		parts[i] = fmt.Sprintf("[%d]", d)
	}
	return "Dice: " + strings.Join(parts, " ")
}

func formatState(state *engine.GameState) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Round: %d/13\n", state.Round)
	fmt.Fprintf(&sb, "Current Player: %s\n", state.CurrentPlayer)
	fmt.Fprintf(&sb, "Phase: %s\n", phaseName(state.Phase))
	fmt.Fprintf(&sb, "Roll Count: %d/3\n", state.RollCount)
	sb.WriteString(formatDice(state.Dice))
	sb.WriteString("\n")
	if len(state.AvailableCategories) > 0 {
		cats := make([]string, len(state.AvailableCategories))
		for i, c := range state.AvailableCategories {
			cats[i] = string(c)
		}
		fmt.Fprintf(&sb, "Available Categories: %s\n", strings.Join(cats, ", "))
	}
	return sb.String()
}

func formatDiceAndState(state *engine.GameState) string {
	var sb strings.Builder
	sb.WriteString(formatDice(state.Dice))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Roll Count: %d/3\n", state.RollCount)
	fmt.Fprintf(&sb, "Phase: %s\n", phaseName(state.Phase))
	if len(state.AvailableCategories) > 0 {
		cats := make([]string, len(state.AvailableCategories))
		for i, c := range state.AvailableCategories {
			cats[i] = string(c)
		}
		fmt.Fprintf(&sb, "Available Categories: %s\n", strings.Join(cats, ", "))
	}
	return sb.String()
}

func formatPlayerScorecard(p engine.PlayerState) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== %s (%s) ===\n", p.Name, p.ID)
	fmt.Fprintf(&sb, "%-18s %s\n", "Category", "Score")
	fmt.Fprintf(&sb, "%-18s %s\n", strings.Repeat("-", 18), strings.Repeat("-", 5))
	for _, c := range engine.AllCategories {
		if p.Scorecard.IsFilled(c) {
			fmt.Fprintf(&sb, "%-18s %5d\n", c, p.Scorecard.GetScore(c))
		} else {
			fmt.Fprintf(&sb, "%-18s %5s\n", c, "-")
		}
	}
	sb.WriteString(strings.Repeat("-", 24) + "\n")
	fmt.Fprintf(&sb, "%-18s %5d\n", "Upper Total", p.Scorecard.UpperTotal())
	if p.Scorecard.HasUpperBonus() {
		fmt.Fprintf(&sb, "%-18s %5d\n", "Upper Bonus", engine.UpperBonusValue)
	}
	fmt.Fprintf(&sb, "%-18s %5d\n", "Total", p.Scorecard.Total())
	return sb.String()
}

func formatFinalScores(state *engine.GameState) string {
	var sb strings.Builder
	sb.WriteString("=== Final Scores ===\n")
	for _, p := range state.Players {
		fmt.Fprintf(&sb, "%s: %d\n", p.Name, p.Scorecard.Total())
	}
	return sb.String()
}

func phaseName(phase engine.GamePhase) string {
	switch phase {
	case engine.PhaseRolling:
		return "Rolling"
	case engine.PhaseChoosing:
		return "Choosing"
	case engine.PhaseFinished:
		return "Finished"
	default:
		return "Waiting"
	}
}
