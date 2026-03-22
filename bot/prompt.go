package bot

import (
	"encoding/json"
	"fmt"
)

func BuildMCPConfig(yatzBinaryPath string) string {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"yatzcli": map[string]interface{}{
				"command": yatzBinaryPath,
				"args":    []string{"mcp"},
			},
		},
	}
	b, _ := json.MarshalIndent(config, "", "  ")
	return string(b)
}

func BuildSystemPrompt(strategy string) string {
	return fmt.Sprintf(`あなたはヤッツィーの対戦プレイヤーです。戦略的にプレイしてください。

戦略:
%s`, strategy)
}

func BuildPrompt(addr, name, strategy string) string {
	return fmt.Sprintf(`ヤッツィーの対戦ゲームに参加してプレイしてください。

手順:
1. join_game で %s に接続（名前: %s）
2. join_game の結果を確認し、自分のターンでなければ wait_for_turn で待機
3. 自分のターンでは: roll_dice → 必要に応じて hold_dice → score
4. score の結果に次のターンの状態が含まれる。wait_for_turn は呼ばないこと。
5. score の結果で Phase が "Finished" なら終了
6. 3-5 を繰り返す

重要: score した後は wait_for_turn を呼んではいけない。score の返り値が既に次のターンの状態。

戦略:
%s

scoreした後はsend_chatで短い実況コメントを日本語で送ること。`, addr, name, strategy)
}
