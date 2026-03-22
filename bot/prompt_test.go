package bot

import (
	"encoding/json"
	"testing"

	"github.com/edge2992/yatzcli/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt("test strategy")
	assert.Contains(t, prompt, "test strategy")
	assert.Contains(t, prompt, "ヤッツィー")
	assert.Contains(t, prompt, "roll")
	assert.Contains(t, prompt, "hold")
	assert.Contains(t, prompt, "score")
}

func TestBuildUserPrompt(t *testing.T) {
	sc1 := engine.NewScorecard()
	sc1.Fill(engine.Ones, 3)

	sc2 := engine.NewScorecard()

	state := &engine.GameState{
		Players: []engine.PlayerState{
			{ID: "player-0", Name: "Human", Scorecard: sc1},
			{ID: "player-1", Name: "Bot", Scorecard: sc2},
		},
		CurrentPlayer:       "player-1",
		Round:               2,
		Dice:                [5]int{3, 3, 5, 2, 6},
		RollCount:           1,
		AvailableCategories: []engine.Category{engine.Twos, engine.Threes, engine.Fours},
	}

	prompt := BuildUserPrompt(state, "player-1")

	assert.Contains(t, prompt, "ラウンド: 2/13")
	assert.Contains(t, prompt, "[3 3 5 2 6]")
	assert.Contains(t, prompt, "ロール: 1/3")
	assert.Contains(t, prompt, "twos:")
	assert.Contains(t, prompt, "threes:")
	assert.Contains(t, prompt, "あなたのスコアカード (Bot)")
	assert.Contains(t, prompt, "相手のスコアカード (Human)")
	assert.Contains(t, prompt, "ones: 3点")
}

func TestBuildRetryPrompt(t *testing.T) {
	state := &engine.GameState{
		Players:             []engine.PlayerState{{ID: "player-0", Name: "P1", Scorecard: engine.NewScorecard()}},
		CurrentPlayer:       "player-0",
		Round:               1,
		Dice:                [5]int{1, 2, 3, 4, 5},
		RollCount:           1,
		AvailableCategories: engine.AllCategories,
	}

	prompt := BuildRetryPrompt(state, "player-0", "category already filled")
	assert.Contains(t, prompt, "category already filled")
	assert.Contains(t, prompt, "別のアクションを選んでください")
}

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, resp *ClaudeResponse)
	}{
		{
			name:  "roll action",
			input: `{"action":"roll","comment":"さあ振るぞ！"}`,
			check: func(t *testing.T, resp *ClaudeResponse) {
				assert.Equal(t, "roll", resp.Action)
				assert.Equal(t, "さあ振るぞ！", resp.Comment)
			},
		},
		{
			name:  "hold action with indices",
			input: `{"action":"hold","indices":[0,1,3],"comment":"3をキープ"}`,
			check: func(t *testing.T, resp *ClaudeResponse) {
				assert.Equal(t, "hold", resp.Action)
				assert.Equal(t, []int{0, 1, 3}, resp.Indices)
			},
		},
		{
			name:  "score action",
			input: `{"action":"score","category":"threes","comment":"3に入れる"}`,
			check: func(t *testing.T, resp *ClaudeResponse) {
				assert.Equal(t, "score", resp.Action)
				assert.Equal(t, "threes", resp.Category)
			},
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:    "empty action",
			input:   `{"action":"","comment":"test"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ParseResponse([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, resp)
		})
	}
}

func TestResponseSchemaJSON(t *testing.T) {
	schemaJSON := ResponseSchemaJSON()
	var schema map[string]interface{}
	err := json.Unmarshal([]byte(schemaJSON), &schema)
	require.NoError(t, err)
	assert.Equal(t, "object", schema["type"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "action")
	assert.Contains(t, props, "indices")
	assert.Contains(t, props, "category")
	assert.Contains(t, props, "comment")
}
