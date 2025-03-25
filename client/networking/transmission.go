package networking

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
)

type messageType int8

const (
	MESSAGE_FORCESWITCH messageType = iota
	MESSAGE_TURNRESOLVE
	MESSAGE_GAMEOVER
	MESSAGE_CONTINUE
)

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
)

type InvalidMsgTypeError struct {
	msgType messageType
}

func (e *InvalidMsgTypeError) Error() string {
	return fmt.Sprintf("tried to decode invalid msg type: %d", e.msgType)
}

type InvalidActionTypeError struct {
	actionName string
}

func (e *InvalidActionTypeError) Error() string {
	return fmt.Sprintf("tried to decode invalid action: %s", e.actionName)
}

func SendData(conn net.Conn, data any) error {
	encoder := gob.NewEncoder(conn)

	err := encoder.Encode(data)
	if err != nil {
		return err
	}

	log.Debug().Msgf("Sent data: %s", reflect.TypeOf(data))

	return nil
}

func AcceptData[T any](conn net.Conn) (T, error) {
	var data T

	decoder := gob.NewDecoder(conn)

	log.Debug().Msgf("waiting for data: %s", reflect.TypeOf(data))
	err := decoder.Decode(&data)
	if err == nil {
		log.Debug().Msgf("got data: %s", reflect.TypeOf(data))
	}

	return data, err
}

func SendAction(conn net.Conn, data state.Action) error {
	concreteName := reflect.TypeOf(data).String()

	// Send concrete type name before the data itself
	_, err := conn.Write([]byte(concreteName + "\n"))
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(conn)
	return encoder.Encode(data)
}

func AcceptAction(conn net.Conn) (state.Action, error) {
	concreteNameBytes := make([]byte, 1024)

	n, err := conn.Read(concreteNameBytes)
	if err != nil {
		return nil, err
	}

	contentString := string(concreteNameBytes[0:n])
	messageParts := strings.Split(contentString, "\n")

	log.Debug().Msgf("action content: %s", contentString)
	log.Debug().Strs("messageParts", messageParts).Msg("")

	concreteName := string(messageParts[0])

	// HACK:
	// Sometimes when reading the concrete type name of the action
	// we end of grabbing the entire message (why? idk thats a good question. it doesn't seem to be consistent).
	// In the case we do grab it, it will be stored in the second part of the message buffer
	// and we set that to be the reader for the decoder rather than the conn itself
	var decoder *gob.Decoder
	if messageParts[1] != "" {
		decoder = gob.NewDecoder(bytes.NewBufferString(messageParts[1]))
	} else {
		decoder = gob.NewDecoder(conn)
	}

	// I tried to use a action type enum here rather than send the string of the concrete type name.
	// However, when sending actions, i only have a pointer to the interface, not the actual concrete type.
	// So I would end up having to do this anyway somewhere else (unless there is a better way to get around gob's lack of support for interfaces)
	switch concreteName {
	case "state.SwitchAction":
		a := &state.SwitchAction{}

		err := decoder.Decode(a)
		return *a, err
	case "state.SkipAction":
		a := &state.SkipAction{}

		err := decoder.Decode(a)
		return *a, err
	case "state.AttackAction":
		a := &state.AttackAction{}

		err := decoder.Decode(a)
		return *a, err
	}

	return nil, &InvalidActionTypeError{concreteName}
}

func SendMessage(conn net.Conn, msgType messageType, msg tea.Msg) error {
	err := binary.Write(conn, binary.LittleEndian, msgType)
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(conn)
	return encoder.Encode(msg)
}

func AcceptMessage(conn net.Conn) (tea.Msg, error) {
	var msgType messageType = -1
	if err := binary.Read(conn, binary.LittleEndian, &msgType); err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(conn)

	switch msgType {
	case MESSAGE_CONTINUE:
		m := &ContinueUpdaterMessage{}

		err := decoder.Decode(m)
		return *m, err
	case MESSAGE_FORCESWITCH:
		m := &ForceSwitchMessage{}

		err := decoder.Decode(m)
		return *m, err
	case MESSAGE_GAMEOVER:
		m := &GameOverMessage{}

		err := decoder.Decode(m)
		return *m, err
	case MESSAGE_TURNRESOLVE:
		m := &TurnResolvedMessage{}

		err := decoder.Decode(m)
		return *m, err
	}

	return nil, &InvalidMsgTypeError{msgType: msgType}
}
