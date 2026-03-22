package cli

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/edge2992/yatzcli/engine"
)

type spectatorState int

const (
	specWatching spectatorState = iota
	specGameOver
)

type specTickMsg struct{}
type specResultMsg engine.AITurnResult
type specDoneMsg struct{ err error }

type spectatorModel struct {
	results    <-chan engine.AITurnResult
	errCh      <-chan error
	players    []engine.BattlePlayer
	speed      time.Duration
	state      spectatorState
	current    *engine.AITurnResult
	history    []engine.AITurnResult
	turnCount  int
	totalTurns int
	err        error
}

// RunSpectator launches the spectator TUI for watching AI battles.
func RunSpectator(
	results <-chan engine.AITurnResult,
	errCh <-chan error,
	players []engine.BattlePlayer,
	speed time.Duration,
) error {
	m := spectatorModel{
		results:    results,
		errCh:      errCh,
		players:    players,
		speed:      speed,
		totalTurns: 13 * len(players),
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m spectatorModel) Init() tea.Cmd {
	return tea.Batch(waitForResult(m.results, m.errCh))
}

func waitForResult(results <-chan engine.AITurnResult, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case result, ok := <-results:
			if !ok {
				// Channel closed — game done. errCh is buffered (cap=1),
				// so this read will not block.
				err := <-errCh
				return specDoneMsg{err: err}
			}
			return specResultMsg(result)
		case err := <-errCh:
			return specDoneMsg{err: err}
		}
	}
}

func specTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return specTickMsg{}
	})
}

func (m spectatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case specResultMsg:
		r := engine.AITurnResult(msg)
		m.current = &r
		m.history = append(m.history, r)
		m.turnCount++
		return m, specTickCmd(m.speed)

	case specTickMsg:
		return m, waitForResult(m.results, m.errCh)

	case specDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.state = specGameOver
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		if m.state == specWatching && m.current != nil {
			// Skip to next result
			return m, waitForResult(m.results, m.errCh)
		}
	}
	return m, nil
}

func (m spectatorModel) View() tea.View {
	var b strings.Builder

	switch m.state {
	case specWatching:
		m.viewWatching(&b)
	case specGameOver:
		m.viewGameOver(&b)
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v\n", m.err))
	}

	return tea.NewView(b.String())
}

func (m spectatorModel) viewWatching(b *strings.Builder) {
	b.WriteString("  === AI Battle ===\n\n")

	if m.current == nil {
		b.WriteString("  Waiting for first turn...\n")
		return
	}

	r := m.current
	b.WriteString(fmt.Sprintf("  Turn %d/%d  |  %s (%s)\n\n",
		m.turnCount, m.totalTurns, r.PlayerName, r.StrategyName))

	// Show hold history if any
	if len(r.HoldHistory) > 0 {
		for i, h := range r.HoldHistory {
			b.WriteString(fmt.Sprintf("  Roll %d: ", i+1))
			for j, d := range h.Dice {
				held := false
				for _, idx := range h.Held {
					if idx == j {
						held = true
						break
					}
				}
				if held {
					b.WriteString(fmt.Sprintf("[*%d*]", d))
				} else {
					b.WriteString(fmt.Sprintf("[ %d ]", d))
				}
				if j < 4 {
					b.WriteString(" ")
				}
			}
			b.WriteString("\n")
		}
	}

	// Final dice
	b.WriteString("  Dice: ")
	for i, d := range r.Dice {
		b.WriteString(fmt.Sprintf("[ %d ]", d))
		if i < 4 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Scored: %-16s  %3d pts\n\n", categoryName(r.Category), r.Score))

	// Scorecard from history
	scorecards, names := m.buildScorecards()
	writeScorecard(b, scorecards, names, false)

	b.WriteString("\n  Press any key to advance, [q] to quit\n")
}

func (m spectatorModel) viewGameOver(b *strings.Builder) {
	b.WriteString("  ===  BATTLE OVER  ===\n\n")

	scorecards, names := m.buildScorecards()
	writeScorecard(b, scorecards, names, true)

	// Winner
	bestScore := -1
	winner := ""
	for _, name := range names {
		sc := scorecards[name]
		total := sc.Total()
		if total > bestScore {
			bestScore = total
			winner = name
		}
	}
	b.WriteString(fmt.Sprintf("\n  Winner: %s with %d points!\n\n", winner, bestScore))
	b.WriteString("  [q] Quit\n")
}

// buildScorecards reconstructs scorecards from turn history.
func (m spectatorModel) buildScorecards() (map[string]*engine.Scorecard, []string) {
	scorecards := make(map[string]*engine.Scorecard)
	names := make([]string, len(m.players))
	for i, p := range m.players {
		sc := engine.NewScorecard()
		scorecards[p.Name] = &sc
		names[i] = p.Name
	}
	for _, r := range m.history {
		if sc, ok := scorecards[r.PlayerName]; ok {
			sc.Fill(r.Category, r.Score)
		}
	}
	return scorecards, names
}

// writeScorecard writes a scorecard table to the builder.
// If showBonus is true, the upper bonus row is included.
func writeScorecard(b *strings.Builder, scorecards map[string]*engine.Scorecard, names []string, showBonus bool) {
	nameWidth := 16
	b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "Category"))
	for _, name := range names {
		b.WriteString(fmt.Sprintf("  %8s", name))
	}
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("-", nameWidth+10*len(names)) + "\n")

	for _, cat := range engine.AllCategories {
		b.WriteString(fmt.Sprintf("  %-*s", nameWidth, categoryName(cat)))
		for _, name := range names {
			sc := scorecards[name]
			if sc.IsFilled(cat) {
				b.WriteString(fmt.Sprintf("  %8d", sc.GetScore(cat)))
			} else {
				b.WriteString(fmt.Sprintf("  %8s", "-"))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("  " + strings.Repeat("-", nameWidth+10*len(names)) + "\n")

	if showBonus {
		b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "Upper Bonus"))
		for _, name := range names {
			sc := scorecards[name]
			if sc.HasUpperBonus() {
				b.WriteString(fmt.Sprintf("  %8d", engine.UpperBonusValue))
			} else {
				b.WriteString(fmt.Sprintf("  %5d/%d", sc.UpperTotal(), engine.UpperBonusThreshold))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("  %-*s", nameWidth, "TOTAL"))
	for _, name := range names {
		sc := scorecards[name]
		b.WriteString(fmt.Sprintf("  %8d", sc.Total()))
	}
	b.WriteString("\n")
}
