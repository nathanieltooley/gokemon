package gameview

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/game/state/stateupdater"
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

const (
	_MESSAGE_TIME     = time.Second * 2
	_TICKS_PER_SECOND = 20
	_TICK_TIME        = 1000 / _TICKS_PER_SECOND
)

type gameContext struct {
	// This state is "The One True State", the actual state that dictates how the game is going
	// If / When multiplayer is added, this will only be true for the host, but it will true be the
	// actual state of that client
	state        *state.GameState
	chosenAction state.Action
}

type MainGameModel struct {
	ctx          *gameContext
	stateUpdater stateupdater.StateUpdater
	playerSide   int
	// Player has submitted their action and starts waiting for the opponent to submit their's
	// and for both of their actions to be processed
	startedTurnResolving bool

	// Intermediate states (in between turns) that need to be displayed to the client
	stateQueue []state.StateUpdate
	// State that should be rendered when inbetween turns
	currentRenderedState state.GameState
	// Whether we started the state update rendering process
	renderingPastState         bool
	waitingOnPastStateMessages bool
	messageQueue               []string
	currentStateMessage        string

	inited bool

	panel tea.Model
}

func NewMainGameModel(state state.GameState, playerSide int) MainGameModel {
	ctx := &gameContext{
		state:        &state,
		chosenAction: nil,
	}

	return MainGameModel{
		ctx:                  ctx,
		playerSide:           playerSide,
		stateUpdater:         &stateupdater.LocalUpdater{},
		currentRenderedState: *ctx.state,
		panel: actionPanel{
			ctx: ctx,
		},
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*_TICK_TIME, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m MainGameModel) Init() tea.Cmd { return nil }

func (m MainGameModel) View() string {
	var stateToRender state.GameState
	stateToRender = m.currentRenderedState

	panelView := ""

	if m.ctx.chosenAction == nil && !m.startedTurnResolving {
		panelView = m.panel.View()
	}

	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,

			fmt.Sprintf("Turn: %d", m.ctx.state.Turn),

			rendering.ButtonStyle.Width(40).Render(m.currentStateMessage),

			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(stateToRender, "HOST", stateToRender.GetPlayer(state.HOST)).View(),
				newPlayerPanel(stateToRender, "PEER", stateToRender.GetPlayer(state.PEER)).View(),
			),

			panelView,
		),
	)
}

type (
	nextNotifMsg struct{}
	nextStateMsg struct{}
)

func (m *MainGameModel) nextState() bool {
	if len(m.stateQueue) != 0 {
		// Pop queue
		stateUpdate := m.stateQueue[0]
		m.currentRenderedState = stateUpdate.State
		m.messageQueue = stateUpdate.Messages

		log.Info().Int("Active Health", int(stateUpdate.State.LocalPlayer.GetActivePokemon().Hp.Value)).Msg("New State: Player")
		log.Info().Int("Active Health", int(stateUpdate.State.OpposingPlayer.GetActivePokemon().Hp.Value)).Msg("New State: AI")

		m.stateQueue = m.stateQueue[1:]

		return true
	}

	return false
}

// Returns true if there was a message in the queue
func (m *MainGameModel) nextStateMsg() bool {
	if len(m.messageQueue) != 0 {
		m.currentStateMessage = m.messageQueue[0]
		m.messageQueue = m.messageQueue[1:]

		log.Info().Msgf("Next Message: %s", m.currentStateMessage)

		return true
	}

	return false
}

// TODO: There will have to be A LOT of changes for LAN or P2P Multiplayer
func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	// Debug switch action
	switch msg := msg.(type) {
	case tea.KeyMsg:
	case nextNotifMsg:
		// For when we still have messages in the queue
		if m.nextStateMsg() {
			cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		} else {
			// Go to the next state once we run out of messages
			if m.nextState() {
				m.nextStateMsg()
				cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
					return nextNotifMsg{}
				}))
			} else {
				// Reset back to normal
				m.currentStateMessage = ""
				m.renderingPastState = false
				m.startedTurnResolving = false
			}
		}
	case tickMsg:
		cmds = append(cmds, tick())

	case stateupdater.ForceSwitchMessage:
		// TODO: Handle Force switch on player side
		if msg.ForThisPlayer {
		} else {
			m.stateQueue = append(m.stateQueue, msg.StateUpdates...)
			cmds = append(cmds, m.stateUpdater.Update(m.ctx.state, true))
		}
	case stateupdater.TurnResolvedMessage:
		m.panel = actionPanel{ctx: m.ctx}
		m.ctx.chosenAction = nil

		m.stateQueue = append(m.stateQueue, msg.StateUpdates...)
	}

	// Force the UI into the switch pokemon panel when the player's current pokemon is dead
	if !m.ctx.state.LocalPlayer.GetActivePokemon().Alive() {
		switch m.panel.(type) {
		case pokemonPanel:
		default:
			m.panel = newPokemonPanel(m.ctx, m.ctx.state.LocalPlayer.Team)
		}
	}

	if m.ctx.chosenAction != nil && !m.startedTurnResolving {
		m.stateUpdater.SendAction(m.ctx.chosenAction)
		// Freezes the state while updates are being made
		m.startedTurnResolving = true
		m.currentRenderedState = *m.ctx.state

		cmds = append(cmds, m.stateUpdater.Update(m.ctx.state, false))
	} else {
		m.panel, _ = m.panel.Update(msg)
	}

	if m.currentStateMessage == "" {
		// If there was a new message, send a cmd to go to the next one
		if m.nextStateMsg() {
			cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		}
	}

	// Once we get some state updates from the state updater,
	// start displaying them
	if len(m.stateQueue) != 0 && !m.renderingPastState {
		m.renderingPastState = true
		m.nextState()
		m.nextStateMsg()

		cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
			return nextNotifMsg{}
		}))
	}

	// Game Over Check
	// NOTE: Assuming singleplayer
	gameOverValue := m.ctx.state.GameOver()
	if gameOverValue != -1 {
		if gameOverValue == m.playerSide {
			return newEndScreen("You Won!"), nil
		} else {
			return newEndScreen("You Lost :("), nil
		}
	}

	// Start tick loop
	if !m.inited {
		cmds = append(cmds, tick())
		m.inited = true
	}

	// Render the actual state before turns get processed
	if !m.startedTurnResolving {
		m.currentRenderedState = *m.ctx.state
	}

	return m, tea.Batch(cmds...)
}
