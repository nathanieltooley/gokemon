package gameview

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/networking"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/rs/zerolog/log"
)

const (
	_MESSAGE_TIME = time.Second * 2
)

var _TICK_TIME = 1000 / global.GameTicksPerSecond

// game state machine
const (
	SM_WAITING_FOR_USER_ACTION = iota
	SM_USER_ACTION_SENT
	SM_WAITING_FOR_OP_ACTION
	SM_RECEIVED_EVENTS
	SM_SHOWING_EVENTS
	SM_ERRORED
)

// Used to send info around different game UI components
type gameContext struct {
	// This state is "The One True State", the actual state that dictates how the game is going
	// If / When multiplayer is added, this will only be true for the host, but it will true be the
	// actual state of that client
	state          *stateCore.GameState
	chosenAction   stateCore.Action
	forcedSwitch   bool
	playerSide     int
	currentSmState int
}

type MainGameModel struct {
	ctx               *gameContext
	turnUpdateHandler func(stateCore.Action) tea.Msg

	// Intermediate states (in between turns) that need to be displayed to the client
	eventQueue stateCore.EventIter
	// Whether we started the state update rendering process
	messageQueue        []string
	currentStateMessage string

	inited bool

	insidePanel bool
	panel       tea.Model

	showError  bool
	currentErr error

	tickCount int64
	netInfo   networking.NetReaderInfo
}

func NewMainGameModel(gameState stateCore.GameState, playerSide int, conn net.Conn) MainGameModel {
	ctx := &gameContext{
		state:          &gameState,
		chosenAction:   nil,
		playerSide:     playerSide,
		currentSmState: SM_WAITING_FOR_USER_ACTION,
	}

	var updater func(stateCore.Action) tea.Msg

	// Buffer size of 1 here since client should not send more than one per turn
	actionChan := make(chan stateCore.Action, 1)
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
			updater = func(action stateCore.Action) tea.Msg {
				return hostNetworkHandler(readerInfo, action, ctx.state)
			}
		case state.PEER:
			updater = func(action stateCore.Action) tea.Msg {
				return clientNetworkHandler(readerInfo, action)
			}
		}
	} else {
		updater = func(action stateCore.Action) tea.Msg {
			return singleplayerHandler(ctx.state, action)
		}
	}

	return MainGameModel{
		ctx:               ctx,
		turnUpdateHandler: updater,
		panel:             newActionPanel(ctx),
		eventQueue:        stateCore.NewEventIter(),

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

	panelView := ""

	if m.ctx.currentSmState == SM_WAITING_FOR_USER_ACTION {
		panelView = m.panel.View()
	}

	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,

			fmt.Sprintf("Turn: %d", m.ctx.state.Turn),

			rendering.ButtonStyle.Width(40).Render(m.currentStateMessage),

			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(*m.ctx.state, m.ctx.state.HostPlayer.Name, m.ctx.state.GetPlayer(state.HOST), &m.ctx.state.HostPlayer.MultiTimerTick).View(),
				// TODO: Randomly generate fun trainer names
				newPlayerPanel(*m.ctx.state, m.ctx.state.ClientPlayer.Name, m.ctx.state.GetPlayer(state.PEER), &m.ctx.state.ClientPlayer.MultiTimerTick).View(),
			),

			panelView,
		),
	)
}

type (
	nextNotifMsg struct{}
	nextStateMsg struct{}
)

// TODO: redo this and nextStateMsg i really don't like how i did these
func (m *MainGameModel) nextEvent() bool {
	messages, ok := m.eventQueue.Next(m.ctx.state)
	if !ok {
		log.Info().Msg("no more events")
		return false
	}

	log.Info().Strs("event messages", messages).Msg("")

	m.messageQueue = append(m.messageQueue, messages...)

	return true
}

// Returns true if there was a message in the queue
func (m *MainGameModel) nextStateMsg() bool {
	if len(m.messageQueue) != 0 {
		m.currentStateMessage = m.messageQueue[0]
		m.messageQueue = m.messageQueue[1:]

		log.Info().Msgf("Rendering next message: %s", m.currentStateMessage)

		return true
	}

	return false
}

func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	if m.showError {
		m.ctx.currentSmState = SM_ERRORED
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
			delay := _MESSAGE_TIME
			cmds = append(cmds, tea.Tick(delay, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		} else {
			// Go to the next event once we run out of messages
			if m.nextEvent() {
				// TODO: Add case for if there is no msg here
				m.nextStateMsg()
				cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
					return nextNotifMsg{}
				}))
			} else {
				// Reset back to normal when we run out of events
				m.currentStateMessage = ""
				m.ctx.currentSmState = SM_WAITING_FOR_USER_ACTION

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
		m.eventQueue.AddEvents(msg.Events)

		if msg.ForThisPlayer {
			m.ctx.chosenAction = nil
			m.ctx.forcedSwitch = true
			m.ctx.currentSmState = SM_WAITING_FOR_USER_ACTION
		} else {
			// Do nothing, it's not our turn to move
			m.ctx.currentSmState = SM_WAITING_FOR_OP_ACTION
			cmds = append(cmds, func() tea.Msg {
				return m.turnUpdateHandler(nil)
			})
		}
	case networking.TurnResolvedMessage:
		log.Debug().Msg("game main has received turn resolved message")
		m.panel = newActionPanel(m.ctx)
		m.ctx.chosenAction = nil
		m.ctx.forcedSwitch = false

		m.ctx.currentSmState = SM_RECEIVED_EVENTS

		for _, event := range msg.Events {
			log.Debug().Str("eventType", reflect.TypeOf(event).Name()).Msg("")
		}

		m.eventQueue.AddEvents(msg.Events)

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
	if m.ctx.currentSmState == SM_USER_ACTION_SENT {
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

		m.ctx.currentSmState = SM_WAITING_FOR_OP_ACTION
	} else {
		m.panel, _ = m.panel.Update(msg)
	}

	// Once we get some state updates from the state updater,
	// start displaying them
	if m.ctx.currentSmState == SM_RECEIVED_EVENTS {
		m.nextEvent()
		m.nextStateMsg()

		cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
			return nextNotifMsg{}
		}))

		m.ctx.currentSmState = SM_SHOWING_EVENTS
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
		m.ctx.chosenAction = stateCore.NewSwitchAction(m.ctx.state, m.ctx.playerSide, playerInfo.ActivePokeIndex)

		m.ctx.currentSmState = SM_USER_ACTION_SENT

		m.inited = true
	}

	return m, tea.Batch(cmds...)
}

func singleplayerHandler(gameState *stateCore.GameState, playerAction stateCore.Action) tea.Msg {
	// Artifical delay
	time.Sleep(time.Second * 2)
	aiAction := state.BestAiAction(gameState)
	// Force AI to switch in on "first" turn on battle as happens in a multiplayer game
	if gameState.Turn == 0 {
		aiAction = stateCore.NewSwitchAction(gameState, state.AI, gameState.ClientPlayer.ActivePokeIndex)
	}
	return state.ProcessTurn(gameState, []stateCore.Action{playerAction, aiAction})
}

func clientNetworkHandler(netInfo networking.NetReaderInfo, action stateCore.Action) tea.Msg {
	log.Debug().Msgf("client action: %+v", action)

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

func hostNetworkHandler(netInfo networking.NetReaderInfo, action stateCore.Action, gameState *stateCore.GameState) tea.Msg {
	opAction, ok := <-netInfo.ActionChan
	if !ok {
		return networking.NetworkingErrorMsg{Err: errors.New("host failed to listen to action channel")}
	}

	actions := []stateCore.Action{action, opAction}
	if action == nil {
		log.Debug().Msg("host's action for this turn is nil, should only happen during force switch")
		actions = []stateCore.Action{opAction}
	}
	turnResult := state.ProcessTurn(gameState, actions)

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
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_FORCESWITCH, networking.ForceSwitchMessage{ForThisPlayer: false, Events: msg.Events})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send force switch message"}
			}
		} else { // for client
			err := networking.SendMessage(netInfo.Conn, networking.MESSAGE_FORCESWITCH, networking.ForceSwitchMessage{ForThisPlayer: true, Events: msg.Events})
			if err != nil {
				return networking.NetworkingErrorMsg{Err: err, Reason: "host failed to send force switch message"}
			}
		}

		return turnResult
	}

	return turnResult
}
