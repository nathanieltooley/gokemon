package networking

import (
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
)

const (
	MESSAGE_FORCESWITCH messageType = iota
	MESSAGE_TURNRESOLVE
	MESSAGE_GAMEOVER
	MESSAGE_CONTINUE
	MESSAGE_SENDACTION
	MESSAGE_UPDATETIMER
)

// "Messages" are during a game for communication
type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		StateUpdates  []state.StateSnapshot
	}
	TurnResolvedMessage struct {
		StateUpdates []state.StateSnapshot
	}
	GameOverMessage struct {
		ForThisPlayer bool
	}
	ContinueUpdaterMessage struct {
		Actions []state.Action
	}
	SendActionMessage struct {
		Action state.Action
	}
	UpdateTimerMessage struct {
		NewTime int
	}
)

// "Packets" are used for networking outside of the game (lobby setup and start)
type TeamSelectionPacket struct {
	Team []game.Pokemon
}

type StarterSelectionPacket struct {
	StartingIndex int
}

type NetworkingErrorMsg struct {
	Err    error
	Reason string
}

func (e NetworkingErrorMsg) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = "error occured while networking"
	}
	return fmt.Sprintf("%s: %s", reason, e.Err)
}

type NetReaderInfo struct {
	ActionChan  chan state.Action
	TimerChan   chan UpdateTimerMessage
	MessageChan chan tea.Msg

	Conn net.Conn
}

func (g NetReaderInfo) CloseChans() { // Doesn't take pointers because channels should be pointer types themselves
	close(g.ActionChan)
	close(g.MessageChan)
	close(g.TimerChan)
}

// Contains all info needed for the current multiplayer connection
type GameNetInfo struct {
	// TODO: Maybe make these send only, would have to think of a work-around for the connection reader though
	ActionChan  <-chan state.Action
	TimerChan   <-chan UpdateTimerMessage
	MessageChan <-chan tea.Msg

	// Should only be used for SENDING messages, reading messages should be done through the channels
	conn net.Conn
}

func NewGameNetInfo(actionChan <-chan state.Action, timerChan <-chan UpdateTimerMessage, messageChan <-chan tea.Msg, conn net.Conn) GameNetInfo {
	return GameNetInfo{
		ActionChan:  actionChan,
		TimerChan:   timerChan,
		MessageChan: messageChan,

		conn: conn,
	}
}

// Wrapper around SendMessage so that the connection is private
func (g GameNetInfo) SendMessage(msgType messageType, msg tea.Msg) error {
	return SendMessage(g.conn, msgType, msg)
}

// Wrapper around SendAction so that the connection is private
func (g GameNetInfo) SendAction(action state.Action) error {
	return SendAction(g.conn, action)
}
