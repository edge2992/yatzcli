package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edge2992/yatzcli/engine"
)

type ClaudeResponse struct {
	Action   string `json:"action"`
	Indices  []int  `json:"indices,omitempty"`
	Category string `json:"category,omitempty"`
	Comment  string `json:"comment"`
}

var responseSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"action":   map[string]interface{}{"type": "string"},
		"indices":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "integer"}},
		"category": map[string]interface{}{"type": "string"},
		"comment":  map[string]interface{}{"type": "string"},
	},
	"required": []string{"action", "comment"},
}

func ResponseSchemaJSON() string {
	b, _ := json.Marshal(responseSchema)
	return string(b)
}

func BuildSystemPrompt(strategy string) string {
	return fmt.Sprintf(`あなたはヤッツィーの対戦プレイヤーです。ゲーム状態を分析し、次のアクションを選んでください。

戦略:
%s

ルール:
- 1ターンに最大3回ロールできる（初回はroll、2-3回目はhold）
- holdではキープするダイスのインデックス(0-4)を指定し、残りをリロール
- 最終的にscoreでカテゴリを選んでスコアする
- 各カテゴリは1回しか使えない

アクション:
- "roll": ターンの最初のロール（rollCountが0の時のみ）
- "hold": キープするダイスのインデックスをindicesで指定してリロール（rollCountが1以上の時）
- "score": categoryでカテゴリを指定してスコアする

レスポンスはJSON形式で返すこと。commentフィールドには日本語で短い実況を入れること。`, strategy)
}

func BuildUserPrompt(state *engine.GameState, playerID string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ラウンド: %d/%d\n", state.Round, engine.MaxRounds))
	sb.WriteString(fmt.Sprintf("ダイス: %v\n", state.Dice))
	sb.WriteString(fmt.Sprintf("ロール: %d/%d\n", state.RollCount, engine.MaxRolls))

	// Available categories with potential scores
	sb.WriteString("利用可能カテゴリと得点:\n")
	for _, cat := range state.AvailableCategories {
		score := engine.CalcScore(cat, state.Dice)
		sb.WriteString(fmt.Sprintf("  %s: %d点\n", cat, score))
	}

	// Both players' scorecards
	for _, p := range state.Players {
		label := "あなた"
		if p.ID != playerID {
			label = "相手"
		}
		sb.WriteString(fmt.Sprintf("\n%sのスコアカード (%s):\n", label, p.Name))
		for _, cat := range engine.AllCategories {
			if p.Scorecard.IsFilled(cat) {
				sb.WriteString(fmt.Sprintf("  %s: %d点\n", cat, p.Scorecard.GetScore(cat)))
			}
		}
		sb.WriteString(fmt.Sprintf("  合計: %d点\n", p.Scorecard.Total()))
	}

	sb.WriteString("\n次のアクションは？")
	return sb.String()
}

func BuildRetryPrompt(state *engine.GameState, playerID string, errMsg string) string {
	return fmt.Sprintf("%s\n\n前回のアクションはエラーになりました: %s\n別のアクションを選んでください。", BuildUserPrompt(state, playerID), errMsg)
}

func ParseResponse(data []byte) (*ClaudeResponse, error) {
	var resp ClaudeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse claude response: %w", err)
	}
	if resp.Action == "" {
		return nil, fmt.Errorf("empty action in response")
	}
	return &resp, nil
}
