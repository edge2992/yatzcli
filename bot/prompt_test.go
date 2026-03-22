package bot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMCPConfig(t *testing.T) {
	config := BuildMCPConfig("/usr/local/bin/yatz")

	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(config), &parsed)
	require.NoError(t, err)

	servers := parsed["mcpServers"].(map[string]interface{})
	yatzcli := servers["yatzcli"].(map[string]interface{})
	assert.Equal(t, "/usr/local/bin/yatz", yatzcli["command"])

	args := yatzcli["args"].([]interface{})
	assert.Equal(t, []interface{}{"mcp"}, args)
}

func TestBuildPrompt(t *testing.T) {
	prompt := BuildPrompt("localhost:9876", "TestBot", "test strategy")

	assert.Contains(t, prompt, "localhost:9876")
	assert.Contains(t, prompt, "TestBot")
	assert.Contains(t, prompt, "test strategy")
	assert.Contains(t, prompt, "join_game")
	assert.Contains(t, prompt, "wait_for_turn")
	assert.Contains(t, prompt, "roll_dice")
	assert.Contains(t, prompt, "score")
	assert.Contains(t, prompt, "send_chat")
	// Critical: instruction not to call wait_for_turn after score
	assert.Contains(t, prompt, "score 後に wait_for_turn を呼ぶな")
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := BuildSystemPrompt("my strategy")
	assert.Contains(t, prompt, "ヤッツィー")
	assert.Contains(t, prompt, "my strategy")
}
