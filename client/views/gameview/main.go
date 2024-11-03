package gameview

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/ai"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/rs/zerolog/log"
)

// BIG PROBLEM IM GONNA HAVE TO FIGURE OUT
//
// bubbletea doesn't have a framework for creating "panels" or "windows" at arbitrary locations.
// I'm gonna have to find a way to atleast put these panels in decent locations.
// Maybe a module that creates these panels at a location and makes a string of it?
// I may not deal with it ever and just make two panels with pokemon info in the center
// and a panel with pokemon actions at the bottom ¯\_(ツ)_/¯

var (
	panelStyle            = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).Padding(1, 2).AlignHorizontal(lipgloss.Center)
	highlightedPanelStyle = panelStyle.Background(rendering.HighlightedColor).Foreground(lipgloss.Color("255"))
)

type MainGameModel struct {
	state      *state.GameState
	playerSide int

	panel tea.Model
}

func NewMainGameModel(state state.GameState, playerSide int) MainGameModel {
	return MainGameModel{
		state:      &state,
		playerSide: playerSide,
		panel: actionPanel{
			state: &state,
		},
	}
}

func (m MainGameModel) Init() tea.Cmd { return nil }
func (m MainGameModel) View() string {
	panelView := ""
	if m.state.LocalSubmittedAction == nil {
		panelView = m.panel.View()
	} else {
		log.Debug().Msg("not your turn")
	}

	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(m.state, "HOST", m.state.GetPlayer(state.HOST)).View(),
				newPlayerPanel(m.state, "PEER", m.state.GetPlayer(state.PEER)).View(),
			),

			panelView,
		),
	)
}

type tickMsg struct {
	t time.Time
}

type refreshOnceMsg struct {
	t time.Time
}

// TODO: There will have to be A LOT of changes for LAN or P2P Multiplayer
func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	receivedOnceRefresh := false

	// Debug switch action
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// if msg.Type == tea.KeyCtrlA {
		// 	m.state.Update(state.SwitchAction{
		// 		PlayerIndex: state.HOST,
		// 		SwitchIndex: 1,
		// 	})
		// }
	case refreshOnceMsg:
		log.Debug().Msgf("Once Refresh Tick: %#v", msg.t.String())
		receivedOnceRefresh = true
	}

	// Force the UI into the switch pokemon panel when the player's current pokemon is dead
	if !m.state.LocalPlayer.GetActivePokemon().Alive() {
		switch m.panel.(type) {
		case pokemonPanel:
		default:
			m.panel = newPokemonPanel(m.state, m.state.LocalPlayer.Team)
		}
	}

	if m.state.LocalSubmittedAction != nil {
		m.state.OpposingSubmittedAction = ai.BestAction(m.state)

		// for now Host goes first
		m.state.LocalSubmittedAction.UpdateState(m.state)

		// Force the AI to update their action if their pokemon died
		// this will have to be expanded for the player and eventually
		// the opposing player instead of just the AI
		switch m.state.OpposingSubmittedAction.(type) {
		case state.AttackAction, state.SkipAction:
			if !m.state.OpposingPlayer.GetActivePokemon().Alive() {
				m.state.OpposingSubmittedAction = ai.BestAction(m.state)
			}
		}

		m.state.OpposingSubmittedAction.UpdateState(m.state)

		m.state.Turn++
		m.state.LocalSubmittedAction = nil
		m.state.OpposingSubmittedAction = nil

		// reset panel to the default after a move is made
		m.panel = actionPanel{state: m.state}

		// Might turn this stuff into a constant tick rate
		// so that the UI is constantly updated
		if !receivedOnceRefresh {
			cmd = tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return refreshOnceMsg{t}
			})
		}
	} else {
		m.panel, _ = m.panel.Update(msg)

		if !receivedOnceRefresh {
			cmd = tea.Tick(time.Second, func(t time.Time) tea.Msg {
				return refreshOnceMsg{t}
			})
		}
	}

	// Game Over Check
	// NOTE: Assuming singleplayer
	gameOverValue := m.state.GameOver()
	if gameOverValue != -1 {
		if gameOverValue == m.playerSide {
			return newEndScreen("You Won!"), nil
		} else {
			return newEndScreen("You Lost :("), nil
		}
	}

	return m, cmd
}
