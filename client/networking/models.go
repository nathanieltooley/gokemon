package networking

import (
	"fmt"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
)

const (
	MESSAGE_FORCESWITCH messageType = iota
	MESSAGE_TURNRESOLVE
	MESSAGE_GAMEOVER
	MESSAGE_CONTINUE
	MESSAGE_SENDACTION
	MESSAGE_UPDATETIMER
)

const (
	DIR_SYNC = iota
	DIR_CLIENT_PAUSE
)

// "Messages" are during a game for communication
type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		Events        []stateCore.StateEvent
	}
	TurnResolvedMessage struct {
		Events []stateCore.StateEvent
	}
	GameOverMessage struct {
		// The "you" in this sense is the player who is receiving the message
		YouLost bool
	}
	ContinueUpdaterMessage struct {
		Actions []stateCore.Action
	}
	SendActionMessage struct {
		Action stateCore.Action
	}
	UpdateTimerMessage struct {
		Directive     int
		NewHostTime   int64
		NewClientTime int64
		HostPaused    bool
		ClientPaused  bool
	}
)

// "Packets" are used for networking outside of the game (lobby setup and start)
type TeamSelectionPacket struct {
	Team []core.Pokemon
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
	ActionChan  chan stateCore.Action
	TimerChan   chan UpdateTimerMessage
	MessageChan chan tea.Msg

	Conn net.Conn
}

func (g NetReaderInfo) CloseChans() { // Doesn't take pointers because channels should be pointer types themselves
	close(g.ActionChan)
	close(g.MessageChan)
	close(g.TimerChan)
}
