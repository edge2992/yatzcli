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
	return fmt.Sprintf(`ヤッツィー対戦プレイヤー。考えすぎず素早くプレイすること。

戦略: %s`, strategy)
}

func BuildPrompt(addr, name, strategy string) string {
	return fmt.Sprintf(`ヤッツィー対戦に参加してプレイせよ。説明や分析は不要。ツールを呼ぶだけでよい。

手順:
1. join_game で %s に接続（名前: %s）
2. 自分のターンでなければ wait_for_turn
3. 自分のターン: まず roll_dice → ダイスを見て hold_dice か score を選ぶ
4. score の返り値が次ターンの状態。score 後に wait_for_turn を呼ぶな（デッドロックする）
5. Phase が "Finished" なら終了。そうでなければ 3 に戻る

ルール:
- 毎ターン最初に必ず roll_dice を呼ぶこと
- hold_dice でキープするダイスのインデックス(0-4)を指定、残りを振り直す
- 最大3回ロール（roll_dice 1回 + hold_dice 最大2回）
- score でカテゴリを選んで得点確定

戦略: %s

score 後に send_chat で一言コメント（日本語、10文字以内）。`, addr, name, strategy)
}
