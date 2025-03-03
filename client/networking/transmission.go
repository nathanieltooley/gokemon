package networking

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/state"
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

	return nil
}

func AcceptData[T any](conn net.Conn) (T, error) {
	decoder := gob.NewDecoder(conn)

	var data T
	err := decoder.Decode(&data)

	return data, err
}

func SendAction(conn net.Conn, data state.Action) error {
	concreteName := reflect.TypeOf(data).String()

	// Send concrete type name before the data itself
	_, err := conn.Write([]byte(concreteName))

	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(conn)
	return encoder.Encode(data)
}

func AcceptAction(conn net.Conn) (state.Action, error) {
	concreteNameBytes := make([]byte, 128)

	n, err := conn.Read(concreteNameBytes)
	if err != nil {
		return nil, err
	}

	concreteName := string(concreteNameBytes[0:n])
	decoder := gob.NewDecoder(conn)

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
