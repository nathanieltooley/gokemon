package gameview

import (
	"errors"
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/game/state/stateupdater"
	"github.com/nathanieltooley/gokemon/client/global"
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
	_MESSAGE_TIME = time.Second * 2
)

var _TICK_TIME = 1000 / global.GameTicksPerSecond

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
	ctx               *gameContext
	turnUpdateHandler func(state.Action) tea.Msg
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

	tickCount int64
	netInfo   networking.NetReaderInfo
}

func NewMainGameModel(gameState state.GameState, playerSide int, conn net.Conn) MainGameModel {
	ctx := &gameContext{
		state:        &gameState,
		chosenAction: nil,
		playerSide:   playerSide,
	}

	var updater func(state.Action) tea.Msg

	// Buffer size of 1 here since client should not send more than one per turn
	actionChan := make(chan state.Action, 1)
	timerChan := make(chan networking.UpdateTimerMessage, 5)
	messageChan := make(chan tea.Msg)

	readerInfo := networking.NetReaderInfo{
		ActionChan:  actionChan,
		TimerChan:   timerChan,
		MessageChan: messageChan,

		Conn: conn,
	}

	if gameState.Networked {
		switch playerSide {
		case state.HOST:
			// maybe changing these from interfaces wasn't the best idea
			updater = func(action state.Action) tea.Msg {
				return hostNetworkHandler(readerInfo, action, ctx.state)
			}
		case state.PEER:
			updater = func(action state.Action) tea.Msg {
				return clientNetworkHandler(readerInfo, action)
			}
		}
	} else {
		updater = func(action state.Action) tea.Msg {
			return singleplayerHandler(ctx.state, action)
		}
	}

	return MainGameModel{
		ctx:                  ctx,
		turnUpdateHandler:    updater,
		currentRenderedState: *ctx.state,
		panel:                newActionPanel(ctx),

		netInfo: readerInfo,
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*time.Duration(_TICK_TIME), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// All reads for a client go through here so multiple msgs could come through around the same time without "race conditions"
// or anything similar
func connectionReader(netInfo networking.NetReaderInfo) tea.Msg {
	for {
		msg, err := networking.AcceptMessage(netInfo.Conn)
		if err != nil {
			log.Err(err).Msg("error while reading from connection")
			netInfo.CloseChans()
			return networking.NetworkingErrorMsg{Err: err, Reason: "error while trying to read connection"}
		}

		switch msg := msg.(type) {
		case networking.UpdateTimerMessage:
			netInfo.TimerChan <- msg
		case networking.SendActionMessage:
			netInfo.ActionChan <- msg.Action
			// Client sent an action so we need to pause their timer
			netInfo.TimerChan <- networking.UpdateTimerMessage{
				Directive: networking.DIR_CLIENT_PAUSE,
			}
		default:
			netInfo.MessageChan <- msg
		}
	}
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

			fmt.Sprintf("Turn: %d", stateToRender.Turn),

			rendering.ButtonStyle.Width(40).Render(m.currentStateMessage),

			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(stateToRender, m.ctx.state.HostPlayer.Name, stateToRender.GetPlayer(state.HOST), &m.ctx.state.HostPlayer.MultiTimerTick).View(),
				// TODO: Randomly generate fun trainer names
				newPlayerPanel(stateToRender, m.ctx.state.ClientPlayer.Name, stateToRender.GetPlayer(state.PEER), &m.ctx.state.ClientPlayer.MultiTimerTick).View(),
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

				// TODO: Move these back to when a turnResolvedMsg is sent when i add skipping of UI msgs
				m.ctx.state.HostPlayer.TimerPaused = false
				m.ctx.state.ClientPlayer.TimerPaused = false
			}
		}
	case tickMsg:
		m.ctx.state.TickPlayerTimers()

		if m.ctx.state.Networked {
			// Host's timer runs out
			if m.ctx.playerSide == state.HOST {
				if m.ctx.state.HostPlayer.MultiTimerTick <= 0 {
					networking.SendMessage(m.netInfo.Conn, networking.MESSAGE_GAMEOVER, networking.GameOverMessage{
						YouLost: false,
					})

					return m, func() tea.Msg {
						return networking.GameOverMessage{
							YouLost: true,
						}
					}
				} else if m.ctx.state.ClientPlayer.MultiTimerTick <= 0 {
					networking.SendMessage(m.netInfo.Conn, networking.MESSAGE_GAMEOVER, networking.GameOverMessage{
						YouLost: true,
					})

					return m, func() tea.Msg {
						return networking.GameOverMessage{
							YouLost: false,
						}
					}
				}
			}

			// send a sync message every second
			if m.ctx.playerSide == state.HOST && m.tickCount%int64(global.GameTicksPerSecond) == 0 {
				_ = networking.SendMessage(m.netInfo.Conn, networking.MESSAGE_UPDATETIMER, networking.UpdateTimerMessage{
					Directive:     networking.DIR_SYNC,
					NewHostTime:   m.ctx.state.HostPlayer.MultiTimerTick,
					NewClientTime: m.ctx.state.ClientPlayer.MultiTimerTick,
					HostPaused:    m.ctx.state.HostPlayer.TimerPaused,
					ClientPaused:  m.ctx.state.ClientPlayer.TimerPaused,
				})
			}
		}

		m.tickCount++

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
			cmds = append(cmds, func() tea.Msg {
				return m.turnUpdateHandler(nil)
			})
		}
	case networking.TurnResolvedMessage:
		m.panel = newActionPanel(m.ctx)
		m.ctx.chosenAction = nil
		m.ctx.forcedSwitch = false

		if m.ctx.playerSide == state.PEER {
			*m.ctx.state = msg.StateUpdates[len(msg.StateUpdates)-1].State.Clone()
		}

		m.stateQueue = append(m.stateQueue, msg.StateUpdates...)

	// Game Over Check
	case networking.GameOverMessage:
		if msg.YouLost {
			return newEndScreen("You Lost :("), nil
		} else {
			return newEndScreen("You Won!"), nil
		}
	case networking.NetworkingErrorMsg:
		m.showError = true
		m.currentErr = msg

		log.Err(msg).Msg("main loop error")

		return m, nil
	}

	// Handle misc msgs and timer sync
	if m.netInfo.Conn != nil {
		select {
		case netMsg := <-m.netInfo.MessageChan:
			// Rather than bring msg handling to a separate function and redo the checks there
			// for any message sent over the wire. Just add it as a cmd and try again next loop
			// NOTE: This may have to change if something is time-sensitive
			cmds = append(cmds, func() tea.Msg {
				return netMsg
			})
		default:
			// No message? Don't care
		}

		select {
		case timerMsg := <-m.netInfo.TimerChan:
			switch timerMsg.Directive {
			case networking.DIR_CLIENT_PAUSE:
				log.Debug().Msg("host told to pause client timer")
				m.ctx.state.ClientPlayer.TimerPaused = true
			case networking.DIR_SYNC:
				// log.Debug().Msgf("client got sync message: %+v", timerMsg)
				client := &m.ctx.state.ClientPlayer
				client.TimerPaused = timerMsg.ClientPaused
				client.MultiTimerTick = int64(timerMsg.NewClientTime)

				host := &m.ctx.state.HostPlayer
				host.TimerPaused = timerMsg.HostPaused
				host.MultiTimerTick = int64(timerMsg.NewHostTime)
			}
		default:
		}
	}

	// Force the UI into the switch pokemon panel when the player's current pokemon is dead
	if !m.ctx.state.HostPlayer.GetActivePokemon().Alive() {
		switch m.panel.(type) {
		case pokemonPanel:
		default:
			m.panel = newPokemonPanel(m.ctx, m.ctx.state.HostPlayer.Team)
		}
	}

	// User has submitted an action
	if m.ctx.chosenAction != nil && !m.startedTurnResolving {
		// Freezes the state while updates are being made
		m.startedTurnResolving = true
		m.currentRenderedState = m.ctx.state.Clone()

		switch m.ctx.playerSide {
		case state.HOST:
			m.ctx.state.HostPlayer.TimerPaused = true
			log.Debug().Msg("host timer should pause")
		case state.PEER:
			m.ctx.state.ClientPlayer.TimerPaused = true
			log.Debug().Msg("client timer should pause")
		}

		cmds = append(cmds, func() tea.Msg {
			return m.turnUpdateHandler(m.ctx.chosenAction)
		})
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

	// Start message reading loops
	if !m.inited {
		cmds = append(cmds, tick())

		if m.ctx.state.Networked {
			cmds = append(cmds, func() tea.Msg {
				return connectionReader(m.netInfo)
			})
		}

		// Make the "First turn" switch ins
		playerInfo := m.ctx.state.GetPlayer(m.ctx.playerSide)
		m.ctx.chosenAction = state.NewSwitchAction(m.ctx.state, m.ctx.playerSide, playerInfo.ActivePokeIndex)

		m.inited = true
	}

	// Render the actual state before turns get processed
	if !m.startedTurnResolving {
		m.currentRenderedState = *m.ctx.state
	}

	return m, tea.Batch(cmds...)
}

func singleplayerHandler(gameState *state.GameState, playerAction state.Action) tea.Msg {
	// Artifical delay
	time.Sleep(time.Second * 2)
	aiAction := state.BestAiAction(gameState)
	return stateupdater.ProcessTurn(gameState, []state.Action{playerAction, aiAction})
}

func clientNetworkHandler(netInfo networking.NetReaderInfo, action state.Action) tea.Msg {
	if action == nil {
		log.Debug().Msg("client is sending action of nil, should only happen during force switch")
	} else {
		err := networking.SendAction(netInfo.Conn, action)
		if err != nil {
			return networking.NetworkingErrorMsg{Err: err, Reason: "client failed to send action to host"}
		}
	}

	return <-netInfo.MessageChan
}

func hostNetworkHandler(netInfo networking.NetReaderInfo, action state.Action, gameState *state.GameState) tea.Msg {
	opAction, ok := <-netInfo.ActionChan
	if !ok {
		return networking.NetworkingErrorMsg{Err: errors.New("host failed to listen to action channel")}
	}

	actions := []state.Action{action, opAction}
	if action == nil {
		log.Debug().Msg("host's action for this turn is nil, should only happen during force switch")
		actions = []state.Action{opAction}
	}
	turnResult := stateupdater.ProcessTurn(gameState, actions)

	switch msg := turnResult.(type) {
	case networking.TurnResolvedMessage:
		err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_TURNRESOLVE, msg)
		if err != nil {
			return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send turn resolve message"}
		}

		return turnResult
	case networking.GameOverMessage:
		// Host Lost, send client a message saying they won
		if msg.YouLost {
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_GAMEOVER, networking.GameOverMessage{YouLost: false})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send game over message"}
			}
		} else { // client lost, send client a message saying they lost
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_GAMEOVER, networking.GameOverMessage{YouLost: true})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send game over message"}
			}
		}
		return turnResult
	case networking.ForceSwitchMessage:
		if msg.ForThisPlayer { // For Host
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_FORCESWITCH, networking.ForceSwitchMessage{ForThisPlayer: false, StateUpdates: msg.StateUpdates})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send force switch message"}
			}
		} else { // for client
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_FORCESWITCH, networking.ForceSwitchMessage{ForThisPlayer: true, StateUpdates: msg.StateUpdates})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send force switch message"}
			}
		}

		return turnResult
	}

	return turnResult
}
