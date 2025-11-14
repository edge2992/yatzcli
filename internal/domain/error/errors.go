package error

import "errors"

// Room関連のエラー
var (
	// ErrRoomNotFound はルームが見つからない場合のエラー
	ErrRoomNotFound = errors.New("room not found")

	// ErrRoomFull はルームが満員の場合のエラー
	ErrRoomFull = errors.New("room is full")

	// ErrGameAlreadyStarted はゲームが既に開始されている場合のエラー
	ErrGameAlreadyStarted = errors.New("game already started")

	// ErrGameNotStarted はゲームが開始されていない場合のエラー
	ErrGameNotStarted = errors.New("game not started")

	// ErrCannotStartGame はゲームを開始できない場合のエラー
	ErrCannotStartGame = errors.New("cannot start game")
)

// Player関連のエラー
var (
	// ErrPlayerNotFound はプレイヤーが見つからない場合のエラー
	ErrPlayerNotFound = errors.New("player not found")

	// ErrPlayerAlreadyInRoom はプレイヤーが既にルームに参加している場合のエラー
	ErrPlayerAlreadyInRoom = errors.New("player already in room")

	// ErrNotPlayerTurn はプレイヤーのターンではない場合のエラー
	ErrNotPlayerTurn = errors.New("not player's turn")

	// ErrInvalidPlayerIndex は無効なプレイヤーインデックスの場合のエラー
	ErrInvalidPlayerIndex = errors.New("invalid player index")
)

// Turn関連のエラー
var (
	// ErrMaxRollsExceeded は最大ロール回数を超えた場合のエラー
	ErrMaxRollsExceeded = errors.New("maximum rolls exceeded")

	// ErrInvalidTurnPhase は無効なターンフェーズの場合のエラー
	ErrInvalidTurnPhase = errors.New("invalid turn phase")

	// ErrInvalidDiceIndex は無効なダイスインデックスの場合のエラー
	ErrInvalidDiceIndex = errors.New("invalid dice index")
)

// Score関連のエラー
var (
	// ErrCategoryAlreadyScored はカテゴリーが既に使用されている場合のエラー
	ErrCategoryAlreadyScored = errors.New("category already scored")

	// ErrInvalidCategory は無効なスコアカテゴリーの場合のエラー
	ErrInvalidCategory = errors.New("invalid score category")

	// ErrInvalidScore は無効なスコア値の場合のエラー
	ErrInvalidScore = errors.New("invalid score value")
)

// Dice関連のエラー
var (
	// ErrInvalidDiceValue はダイスの値が1-6の範囲外の場合のエラー
	ErrInvalidDiceValue = errors.New("dice value must be 1-6")
)

// 汎用エラー
var (
	// ErrInvalidInput は無効な入力の場合のエラー
	ErrInvalidInput = errors.New("invalid input")

	// ErrInternalError は内部エラーの場合のエラー
	ErrInternalError = errors.New("internal error")
)
