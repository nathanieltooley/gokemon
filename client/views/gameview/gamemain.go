package gameview

import (
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/game/state/stateupdater"
	"github.com/nathanieltooley/gokemon/client/networking"
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

const (
	_MESSAGE_TIME     = time.Second * 2
	_TICKS_PER_SECOND = 20
	_TICK_TIME        = 1000 / _TICKS_PER_SECOND
)

// Used to send info around different game UI components
type gameContext struct {
	// This state is "The One True State", the actual state that dictates how the game is going
	// If / When multiplayer is added, this will only be true for the host, but it will true be the
	// actual state of that client
	state        *state.GameState
	chosenAction state.Action
	forcedSwitch bool
	playerSide   int
}

type MainGameModel struct {
	ctx          *gameContext
	stateUpdater stateupdater.StateUpdater
	// Player has submitted their action and starts waiting for the opponent to submit their's
	// and for both of their actions to be processed
	startedTurnResolving bool

	// Intermediate states (in between turns) that need to be displayed to the client
	stateQueue []state.StateSnapshot
	// State that should be rendered when inbetween turns
	currentRenderedState state.GameState
	// Whether we started the state update rendering process
	renderingPastState         bool
	waitingOnPastStateMessages bool
	messageQueue               []string
	currentStateMessage        string

	inited bool

	insidePanel bool
	panel       tea.Model

	showError  bool
	currentErr error

	conn net.Conn
}

func NewMainGameModel(gameState state.GameState, playerSide int, conn net.Conn) MainGameModel {
	ctx := &gameContext{
		state:        &gameState,
		chosenAction: nil,
		playerSide:   playerSide,
	}

	var updater stateupdater.StateUpdater = stateupdater.LocalUpdater

	if conn != nil {
		switch playerSide {
		case state.HOST:
			// maybe changing these from interfaces wasn't the best idea
			updater = func(gs *state.GameState, a []state.Action) tea.Cmd {
				return stateupdater.NetHostUpdater(gs, a, conn)
			}
		case state.PEER:
			updater = func(gs *state.GameState, a []state.Action) tea.Cmd {
				return stateupdater.NetClientUpdater(gs, a, conn)
			}
		}
	}

	return MainGameModel{
		ctx:                  ctx,
		stateUpdater:         updater,
		currentRenderedState: *ctx.state,
		panel:                newActionPanel(ctx),
		conn:                 conn,
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
	if m.showError {
		errorStyle := lipgloss.NewStyle().Border(lipgloss.BlockBorder(), true)
		return rendering.GlobalCenter(errorStyle.Render(lipgloss.JoinVertical(lipgloss.Center, "Error", m.currentErr.Error())))
	}

	stateToRender := m.currentRenderedState

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
				newPlayerPanel(stateToRender, "You", stateToRender.GetPlayer(state.HOST)).View(),
				// TODO: Randomly generate fun trainer names
				newPlayerPanel(stateToRender, "AI", stateToRender.GetPlayer(state.PEER)).View(),
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

	if m.showError {
		return m, nil
	}

	switch m.panel.(type) {
	case actionPanel:
		m.insidePanel = false
	default:
		m.insidePanel = true
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			if m.insidePanel {
				m.panel = newActionPanel(m.ctx)
			}
		}
	case nextNotifMsg:
		// For when we still have messages in the queue
		if m.nextStateMsg() {
			cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		} else {
			// Go to the next state once we run out of messages
			if m.nextState() {
				// TODO: Add case for if there is no msg here
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

	case networking.ForceSwitchMessage:
		if msg.ForThisPlayer {
			m.ctx.chosenAction = nil
			// m.startedTurnResolving = false
			m.ctx.forcedSwitch = true
			// m.panel = newPokemonPanel(m.ctx, m.ctx.state.LocalPlayer.Team)

			m.stateQueue = append(m.stateQueue, msg.StateUpdates...)
		} else {
			m.stateQueue = append(m.stateQueue, msg.StateUpdates...)
			// Do nothing, it's not our turn to move
			cmds = append(cmds, m.stateUpdater(m.ctx.state, nil))
		}
	case networking.TurnResolvedMessage:
		m.panel = newActionPanel(m.ctx)
		m.ctx.chosenAction = nil
		m.ctx.forcedSwitch = false

		m.stateQueue = append(m.stateQueue, msg.StateUpdates...)

	// Game Over Check
	// NOTE: Assuming singleplayer
	case networking.GameOverMessage:
		if msg.ForThisPlayer {
			return newEndScreen("You Won!"), nil
		} else {
			return newEndScreen("You Lost :("), nil
		}
	case stateupdater.NetworkingErrorMsg:
		m.showError = true
		m.currentErr = msg

		log.Err(msg).Msg("main loop error")

		return m, nil
	}

	// Force the UI into the switch pokemon panel when the player's current pokemon is dead
	if !m.ctx.state.LocalPlayer.GetActivePokemon().Alive() {
		switch m.panel.(type) {
		case pokemonPanel:
		default:
			m.panel = newPokemonPanel(m.ctx, m.ctx.state.LocalPlayer.Team)
		}
	}

	// User has submitted an action
	if m.ctx.chosenAction != nil && !m.startedTurnResolving {
		// Freezes the state while updates are being made
		m.startedTurnResolving = true
		m.currentRenderedState = m.ctx.state.Clone()

		cmds = append(cmds, m.stateUpdater(m.ctx.state, []state.Action{m.ctx.chosenAction}))
	} else {
		m.panel, _ = m.panel.Update(msg)
	}

	// Start notification loop when the current message is empty
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
