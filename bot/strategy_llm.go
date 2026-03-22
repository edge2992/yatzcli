package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/edge2992/yatzcli/engine"
)

// LLMStrategy implements engine.Strategy using the Claude API.
type LLMStrategy struct {
	client  anthropic.Client
	model   string
	persona *Persona
}

// NewLLMStrategy creates a new LLM-based strategy.
// If apiKey is empty, the SDK reads ANTHROPIC_API_KEY from the environment.
func NewLLMStrategy(apiKey, model string, persona *Persona) *LLMStrategy {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	client := anthropic.NewClient(opts...)

	if persona == nil {
		persona = &Persona{
			Name:     "LLM",
			Strategy: DefaultStrategy,
		}
	}

	return &LLMStrategy{
		client:  client,
		model:   model,
		persona: persona,
	}
}

func (s *LLMStrategy) Name() string {
	return "llm:" + s.persona.Name
}

func (s *LLMStrategy) DecideAction(dice [5]int, rollCount int, scorecard engine.Scorecard, available []engine.Category) engine.TurnAction {
	action, err := s.callAPI(dice, rollCount, scorecard, available)
	if err != nil {
		// Fallback to greedy on error
		greedy := &engine.GreedyStrategy{}
		return greedy.DecideAction(dice, rollCount, scorecard, available)
	}
	return action
}

type llmResponse struct {
	Action    string `json:"action"`
	Indices   []int  `json:"indices"`
	Category  string `json:"category"`
	Reasoning string `json:"reasoning"`
}

func (s *LLMStrategy) callAPI(dice [5]int, rollCount int, scorecard engine.Scorecard, available []engine.Category) (engine.TurnAction, error) {
	systemPrompt := s.buildSystemPrompt()
	userPrompt := s.buildUserPrompt(dice, rollCount, scorecard, available)

	resp, err := s.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     s.model,
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(userPrompt),
			),
		},
	})
	if err != nil {
		return engine.TurnAction{}, fmt.Errorf("API call failed: %w", err)
	}

	// Extract text from response
	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	return s.parseResponse(responseText, available)
}

func (s *LLMStrategy) buildSystemPrompt() string {
	var b strings.Builder
	b.WriteString("あなたはヤッツィー（Yahtzee）のプレイヤーです。\n\n")

	if s.persona.Personality != "" {
		b.WriteString("## あなたの性格\n")
		b.WriteString(s.persona.Personality)
		b.WriteString("\n\n")
	}
	if s.persona.Strategy != "" {
		b.WriteString("## あなたの戦略\n")
		b.WriteString(s.persona.Strategy)
		b.WriteString("\n\n")
	}
	if s.persona.Catchphrase != "" {
		b.WriteString("## 口癖\n")
		b.WriteString(s.persona.Catchphrase)
		b.WriteString("\n\n")
	}

	b.WriteString(`## ルール
- 5つのダイスを振り、最大3回まで振り直せる（1回目は自動ロール）
- holdで指定したダイスをキープし、残りを振り直す
- 13カテゴリから1つ選んでスコアする
- 上段（ones〜sixes）合計63以上で35点ボーナス

## カテゴリ
ones, twos, threes, fours, fives, sixes: 対応する目の合計
three_of_a_kind: 同じ目3つ以上→全ダイスの合計
four_of_a_kind: 同じ目4つ以上→全ダイスの合計
full_house: 3+2の組み合わせ→25点
small_straight: 4連続→30点
large_straight: 5連続→40点
yahtzee: 全て同じ目→50点
chance: 全ダイスの合計

## 出力形式
以下のJSON形式のみで回答してください。他の文章は不要です。
{"action":"hold"|"score", "indices":[0-4のインデックス配列], "category":"カテゴリ名", "reasoning":"理由"}

- holdの場合: indicesにキープするダイスのインデックスを指定
- scoreの場合: categoryにスコアするカテゴリ名を指定
- 3回目のロール（rollCount=3）の場合は必ずscoreを選択
`)

	return b.String()
}

func (s *LLMStrategy) buildUserPrompt(dice [5]int, rollCount int, scorecard engine.Scorecard, available []engine.Category) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("ダイス: [%d, %d, %d, %d, %d]\n", dice[0], dice[1], dice[2], dice[3], dice[4]))
	b.WriteString(fmt.Sprintf("ロール回数: %d/3\n", rollCount))

	b.WriteString("利用可能カテゴリ:\n")
	for _, c := range available {
		score := engine.CalcScore(c, dice)
		b.WriteString(fmt.Sprintf("  %s: %d点\n", string(c), score))
	}

	b.WriteString("記入済みカテゴリ:\n")
	for _, c := range engine.AllCategories {
		if scorecard.IsFilled(c) {
			b.WriteString(fmt.Sprintf("  %s: %d点\n", string(c), scorecard.GetScore(c)))
		}
	}

	b.WriteString(fmt.Sprintf("上段合計: %d/63\n", scorecard.UpperTotal()))

	if rollCount >= engine.MaxRolls {
		b.WriteString("\n3回目のロール済みです。必ずscoreを選択してください。\n")
	}

	return b.String()
}

func (s *LLMStrategy) parseResponse(text string, available []engine.Category) (engine.TurnAction, error) {
	text = strings.TrimSpace(text)

	// Find JSON in the response (may be wrapped in markdown code blocks)
	jsonStr := text
	if idx := strings.Index(text, "{"); idx >= 0 {
		end := strings.LastIndex(text, "}")
		if end > idx {
			jsonStr = text[idx : end+1]
		}
	}

	var resp llmResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return engine.TurnAction{}, fmt.Errorf("failed to parse LLM response: %w\nraw: %s", err, text)
	}

	switch resp.Action {
	case "hold":
		for _, idx := range resp.Indices {
			if idx < 0 || idx > 4 {
				return engine.TurnAction{}, fmt.Errorf("invalid hold index: %d", idx)
			}
		}
		return engine.TurnAction{
			Type:    "hold",
			Indices: resp.Indices,
		}, nil

	case "score":
		cat := engine.Category(resp.Category)
		valid := false
		for _, c := range available {
			if c == cat {
				valid = true
				break
			}
		}
		if !valid {
			return engine.TurnAction{}, fmt.Errorf("invalid category %q: not available", resp.Category)
		}
		return engine.TurnAction{
			Type:     "score",
			Category: cat,
		}, nil

	default:
		return engine.TurnAction{}, fmt.Errorf("unknown action %q", resp.Action)
	}
}
