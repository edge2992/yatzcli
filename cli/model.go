package cli

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/edge2992/yatzcli/engine"
)

// ChatEntry is a generic chat message for the TUI (no dependency on p2p).
type ChatEntry struct {
	Name string
	Text string
}

type chatMsg ChatEntry

type aiTickMsg struct{}

type uiState int

const (
	stateRolling  uiState = iota
	stateChoosing
	stateWaiting
	stateShowingAI
	stateGameOver
)

type scoreResultMsg struct {
	state *engine.GameState
	err   error
}

type stateUpdateMsg struct {
	state *engine.GameState
}

type model struct {
	client         engine.GameClient
	playerName     string
	playerID       string
	state          uiState
	held           [5]bool
	cursor         int
	lastState      *engine.GameState
	err            string
	opponentStatus string
	chatMessages   []ChatEntry
	chatCh         <-chan ChatEntry
	stateUpdateCh  <-chan *engine.GameState
	aiResults      []engine.AITurnResult
	aiResultIndex  int
}

func newModel(client engine.GameClient, playerName string) model {
	s, _ := client.GetState()
	// Find player ID by name
	playerID := ""
	if s != nil {
		for _, p := range s.Players {
			if p.Name == playerName {
				playerID = p.ID
				break
			}
		}
	}
	return model{
		client:     client,
		playerName: playerName,
		playerID:   playerID,
		lastState:  s,
	}
}

func listenForChat(chatCh <-chan ChatEntry) tea.Cmd {
	return func() tea.Msg {
		ce, ok := <-chatCh
		if !ok {
			return nil
		}
		return chatMsg(ce)
	}
}

func listenForStateUpdate(ch <-chan *engine.GameState) tea.Cmd {
	return func() tea.Msg {
		gs, ok := <-ch
		if !ok {
			return nil
		}
		return stateUpdateMsg{state: gs}
	}
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.chatCh != nil {
		cmds = append(cmds, listenForChat(m.chatCh))
	}
	if m.stateUpdateCh != nil {
		cmds = append(cmds, listenForStateUpdate(m.stateUpdateCh))
	}
	return tea.Batch(cmds...)
}

func aiTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return aiTickMsg{}
	})
}

func (m model) enterAIShowOrNext() (model, tea.Cmd) {
	if lc, ok := m.client.(*engine.LocalClient); ok && len(lc.LastAIResults) > 0 {
		m.aiResults = lc.LastAIResults
		lc.LastAIResults = nil
		m.aiResultIndex = 0
		m.state = stateShowingAI
		return m, aiTickCmd()
	}
	if m.lastState.Phase == engine.PhaseFinished {
		m.state = stateGameOver
	} else {
		m.state = stateRolling
	}
	return m, nil
}

func (m model) advanceAIResult() (model, tea.Cmd) {
	m.aiResultIndex++
	if m.aiResultIndex >= len(m.aiResults) {
		m.aiResults = nil
		m.aiResultIndex = 0
		if m.lastState.Phase == engine.PhaseFinished {
			m.state = stateGameOver
		} else {
			m.state = stateRolling
		}
		return m, nil
	}
	return m, aiTickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case chatMsg:
		m.chatMessages = append(m.chatMessages, ChatEntry(msg))
		if len(m.chatMessages) > 5 {
			m.chatMessages = m.chatMessages[len(m.chatMessages)-5:]
		}
		if m.chatCh != nil {
			return m, listenForChat(m.chatCh)
		}
		return m, nil
	case stateUpdateMsg:
		m.lastState = msg.state
		if msg.state.Phase == engine.PhaseFinished {
			m.state = stateGameOver
			m.opponentStatus = ""
		} else if m.state == stateWaiting && msg.state.CurrentPlayer == m.playerID {
			// Our turn now
			m.state = stateRolling
			m.held = [5]bool{}
			m.opponentStatus = ""
		} else if m.state == stateWaiting {
			// Opponent is playing — show what they're doing
			switch {
			case msg.state.RollCount == 0:
				m.opponentStatus = "考え中..."
			case msg.state.Phase == engine.PhaseChoosing:
				m.opponentStatus = fmt.Sprintf("ダイス確定 [%s] → カテゴリ選択中...", formatDiceCompact(msg.state.Dice))
			default:
				m.opponentStatus = fmt.Sprintf("ロール %d/%d [%s]", msg.state.RollCount, engine.MaxRolls, formatDiceCompact(msg.state.Dice))
			}
		}
		if m.stateUpdateCh != nil {
			return m, listenForStateUpdate(m.stateUpdateCh)
		}
		return m, nil
	case scoreResultMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = stateRolling
			return m, nil
		}
		m.lastState = msg.state
		m.held = [5]bool{}
		return m.enterAIShowOrNext()
	case aiTickMsg:
		if m.state == stateShowingAI {
			return m.advanceAIResult()
		}
	case tea.KeyPressMsg:
		m.err = ""
		switch m.state {
		case stateRolling:
			return m.updateRolling(msg)
		case stateChoosing:
			return m.updateChoosing(msg)
		case stateWaiting:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case stateShowingAI:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m.advanceAIResult()
		case stateGameOver:
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) updateRolling(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r":
		var gs *engine.GameState
		var err error
		if m.lastState.RollCount == 0 {
			gs, err = m.client.Roll()
		} else {
			indices := m.heldIndices()
			gs, err = m.client.Hold(indices)
		}
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.lastState = gs
		if gs.Phase == engine.PhaseChoosing {
			m.state = stateChoosing
			m.cursor = 0
		}
		return m, nil
	case "1", "2", "3", "4", "5":
		if m.lastState.RollCount > 0 {
			idx := int(msg.String()[0]-'0') - 1
			m.held[idx] = !m.held[idx]
		}
		return m, nil
	case "s":
		if m.lastState.RollCount > 0 {
			m.state = stateChoosing
			m.cursor = 0
		}
		return m, nil
	}
	return m, nil
}

func (m model) updateChoosing(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	avail := m.lastState.AvailableCategories
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.lastState.Phase == engine.PhaseRolling {
			m.state = stateRolling
		}
		return m, nil
	case "j", "down":
		if m.cursor < len(avail)-1 {
			m.cursor++
		}
		return m, nil
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "enter":
		if len(avail) == 0 {
			return m, nil
		}
		cat := avail[m.cursor]
		m.state = stateWaiting
		client := m.client
		return m, func() tea.Msg {
			gs, err := client.Score(cat)
			return scoreResultMsg{state: gs, err: err}
		}
	}
	return m, nil
}

func (m model) heldIndices() []int {
	var indices []int
	for i, h := range m.held {
		if h {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m model) View() tea.View {
	if m.lastState == nil {
		return tea.NewView("Loading...")
	}

	var b strings.Builder

	switch m.state {
	case stateRolling:
		m.viewRolling(&b)
	case stateChoosing:
		m.viewChoosing(&b)
	case stateWaiting:
		m.viewWaiting(&b)
	case stateShowingAI:
		m.viewShowingAI(&b)
	case stateGameOver:
		m.viewGameOver(&b)
	}

	m.viewChat(&b)

	if m.err != "" {
		b.WriteString(fmt.Sprintf("\n  Error: %s\n", m.err))
	}

	return tea.NewView(b.String())
}

func (m model) viewRolling(b *strings.Builder) {
	gs := m.lastState
	b.WriteString(fmt.Sprintf("  Round %d/13  |  Player: %s  |  Rolls: %d/%d\n\n",
		gs.Round, m.currentPlayerName(), gs.RollCount, engine.MaxRolls))

	m.viewDice(b)
	b.WriteString("\n")
	m.viewScorecard(b)
	b.WriteString("\n")

	if gs.RollCount == 0 {
		b.WriteString("  [r] Roll dice  [q] Quit\n")
	} else {
		b.WriteString("  [r] Reroll  [1-5] Toggle hold  [s] Score  [q] Quit\n")
	}
}

func (m model) viewChoosing(b *strings.Builder) {
	gs := m.lastState
	b.WriteString(fmt.Sprintf("  Round %d/13  |  Player: %s  |  Choose a category\n\n",
		gs.Round, m.currentPlayerName()))

	m.viewDice(b)
	b.WriteString("\n")

	avail := gs.AvailableCategories
	b.WriteString("  Available categories:\n\n")
	for i, cat := range avail {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		score := engine.CalcScore(cat, gs.Dice)
		b.WriteString(fmt.Sprintf("  %s%-16s  %3d pts\n", cursor, categoryName(cat), score))
	}
	b.WriteString("\n")

	m.viewScorecard(b)
	b.WriteString("\n")

	if gs.Phase == engine.PhaseRolling {
		b.WriteString("  [j/k] Move  [enter] Select  [esc] Back  [q] Quit\n")
	} else {
		b.WriteString("  [j/k] Move  [enter] Select  [q] Quit\n")
	}
}

func (m model) viewWaiting(b *strings.Builder) {
	gs := m.lastState
	// Find opponent name
	opponent := "opponent"
	for _, p := range gs.Players {
		if p.ID == gs.CurrentPlayer && p.Name != m.playerName {
			opponent = p.Name
		}
	}
	b.WriteString(fmt.Sprintf("  Round %d/13  |  %s のターンを待っています...\n\n", gs.Round, opponent))
	if m.opponentStatus != "" {
		b.WriteString(fmt.Sprintf("  ▶ %s\n\n", m.opponentStatus))
	}
	m.viewDice(b)
	b.WriteString("\n")
	m.viewScorecard(b)
	b.WriteString("\n")
	b.WriteString("  [q] Quit\n")
}

func (m model) viewGameOver(b *strings.Builder) {
	b.WriteString("  ===  GAME OVER  ===\n\n")
	m.viewScorecard(b)
	b.WriteString("\n")

	winner := m.lastState.Players[0]
	for _, p := range m.lastState.Players[1:] {
		if p.Scorecard.Total() > winner.Scorecard.Total() {
			winner = p
		}
	}
	b.WriteString(fmt.Sprintf("  Winner: %s with %d points!\n\n", winner.Name, winner.Scorecard.Total()))
	b.WriteString("  [q] Quit\n")
}

func (m model) viewChat(b *strings.Builder) {
	if len(m.chatMessages) == 0 {
		return
	}
	b.WriteString("\n  ─── Chat ────────────────────────\n")
	for _, c := range m.chatMessages {
		b.WriteString(fmt.Sprintf("  %s: %s\n", c.Name, c.Text))
	}
}

func (m model) viewShowingAI(b *strings.Builder) {
	if m.aiResultIndex >= len(m.aiResults) {
		return
	}
	r := m.aiResults[m.aiResultIndex]
	b.WriteString(fmt.Sprintf("  === %s's Turn ===\n\n", r.PlayerName))
	b.WriteString("  Dice: ")
	for i, d := range r.Dice {
		b.WriteString(fmt.Sprintf("[ %d ]", d))
		if i < 4 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Scored: %-16s  %3d pts\n\n", categoryName(r.Category), r.Score))
	b.WriteString(fmt.Sprintf("  (%d/%d)  Press any key to continue...\n", m.aiResultIndex+1, len(m.aiResults)))
}

func (m model) viewDice(b *strings.Builder) {
	gs := m.lastState
	if gs.RollCount == 0 {
		b.WriteString("  Dice: [ - ] [ - ] [ - ] [ - ] [ - ]\n")
		return
	}
	b.WriteString("  Dice: ")
	for i, d := range gs.Dice {
		if m.held[i] {
			b.WriteString(fmt.Sprintf("[*%d*]", d))
		} else {
			b.WriteString(fmt.Sprintf("[ %d ]", d))
		}
		if i < 4 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
	b.WriteString("         ")
	for i := range gs.Dice {
		if m.held[i] {
			b.WriteString(" held")
		} else {
			b.WriteString("    " + fmt.Sprintf("%d", i+1))
		}
		if i < 4 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
}

func (m model) viewScorecard(b *strings.Builder) {
	gs := m.lastState
	players := gs.Players

	nameWidth := 16
	b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "Category"))
	for _, p := range players {
		b.WriteString(fmt.Sprintf("  %8s", p.Name))
	}
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("-", nameWidth+10*len(players)) + "\n")

	for _, cat := range engine.AllCategories {
		b.WriteString(fmt.Sprintf("  %-*s", nameWidth, categoryName(cat)))
		for _, p := range players {
			if p.Scorecard.IsFilled(cat) {
				b.WriteString(fmt.Sprintf("  %8d", p.Scorecard.GetScore(cat)))
			} else {
				b.WriteString(fmt.Sprintf("  %8s", "-"))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("  " + strings.Repeat("-", nameWidth+10*len(players)) + "\n")
	b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "Upper Bonus"))
	for _, p := range players {
		if p.Scorecard.HasUpperBonus() {
			b.WriteString(fmt.Sprintf("  %8d", engine.UpperBonusValue))
		} else {
			ut := p.Scorecard.UpperTotal()
			b.WriteString(fmt.Sprintf("  %5d/%d", ut, engine.UpperBonusThreshold))
		}
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "TOTAL"))
	for _, p := range players {
		b.WriteString(fmt.Sprintf("  %8d", p.Scorecard.Total()))
	}
	b.WriteString("\n")
}

func (m model) currentPlayerName() string {
	for _, p := range m.lastState.Players {
		if p.ID == m.lastState.CurrentPlayer {
			return p.Name
		}
	}
	return m.lastState.CurrentPlayer
}

func categoryName(c engine.Category) string {
	names := map[engine.Category]string{
		engine.Ones:          "Ones",
		engine.Twos:          "Twos",
		engine.Threes:        "Threes",
		engine.Fours:         "Fours",
		engine.Fives:         "Fives",
		engine.Sixes:         "Sixes",
		engine.ThreeOfAKind:  "Three of a Kind",
		engine.FourOfAKind:   "Four of a Kind",
		engine.FullHouse:     "Full House",
		engine.SmallStraight: "Small Straight",
		engine.LargeStraight: "Large Straight",
		engine.Yahtzee:       "Yahtzee",
		engine.Chance:        "Chance",
	}
	if name, ok := names[c]; ok {
		return name
	}
	return string(c)
}

func formatDiceCompact(dice [5]int) string {
	return fmt.Sprintf("%d %d %d %d %d", dice[0], dice[1], dice[2], dice[3], dice[4])
}
