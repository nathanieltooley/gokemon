package networking

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/rs/zerolog/log"
)

type messageType int8

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

// Sends an action message (SendActionMessage)
func SendAction(conn net.Conn, data stateCore.Action) error {
	concreteName := reflect.TypeOf(data).String()

	buffer := bytes.NewBuffer(make([]byte, 0))

	// Write message type first
	if err := binary.Write(buffer, binary.LittleEndian, MESSAGE_SENDACTION); err != nil {
		return err
	}

	// Send concrete type name next
	_, err := buffer.Write([]byte(concreteName + "\n"))
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(buffer)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	_, err = conn.Write(buffer.Bytes())
	return err
}

// Accepts an action AFTER the message type has already been read and removed from the connection
func acceptAction(reader io.Reader) (stateCore.Action, error) {
	actionBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	messageParts := bytes.SplitN(actionBytes, []byte{'\n'}, 2)

	log.Debug().Msgf("action content: %s", actionBytes)

	// Print out all parts of the message
	messagePartsEvent := log.Debug().Int("messageParts", len(messageParts))
	for i, msg := range messageParts {
		messagePartsEvent = messagePartsEvent.Bytes(fmt.Sprintf("messagePart%d", i), msg)
	}
	messagePartsEvent.Msg("")

	concreteName := string(messageParts[0])
	actionContentBuffer := bytes.NewBuffer(messageParts[1])

	decoder := gob.NewDecoder(actionContentBuffer)

	// I tried to use a action type enum here rather than send the string of the concrete type name.
	// However, when sending actions, i only have a pointer to the interface, not the actual concrete type.
	// So I would end up having to do this anyway somewhere else (unless there is a better way to get around gob's lack of support for interfaces)
	switch concreteName {
	case "state.SwitchAction":
		a := &stateCore.SwitchAction{}

		err := decoder.Decode(a)
		return *a, err
	case "state.SkipAction":
		a := &stateCore.SkipAction{}

		err := decoder.Decode(a)
		return *a, err
	case "state.AttackAction":
		a := &stateCore.AttackAction{}

		err := decoder.Decode(a)
		return *a, err
	}

	return nil, &InvalidActionTypeError{concreteName}
}

func SendMessage(conn net.Conn, msgType messageType, msg tea.Msg) error {
	buffer := bytes.NewBuffer(make([]byte, 0))
	err := binary.Write(buffer, binary.LittleEndian, msgType)
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(buffer)
	if err := encoder.Encode(msg); err != nil {
		return err
	}

	_, err = conn.Write(buffer.Bytes())
	return err
}

func AcceptMessage(conn net.Conn) (tea.Msg, error) {
	var msgType messageType = -1

	// TODO: Resolving messages can be VERY large!!!
	// Trim down the size of state!!!
	// Size for two pokemon is about 11kb so times 6 is 66kb round up to 70kb
	// readBytes := make([]byte, 1024*70)
	// n, err := conn.Read(readBytes)
	// if err != nil {
	// 	return nil, err
	// }

	readBytes, err := readAll(conn)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("Message size: %d", len(readBytes))

	buffer := bytes.NewReader(readBytes)

	if err := binary.Read(buffer, binary.LittleEndian, &msgType); err != nil {
		return nil, err
	}

	switch msgType {
	case MESSAGE_CONTINUE:
		return decodeMessage[ContinueUpdaterMessage](buffer)
	case MESSAGE_FORCESWITCH:
		return decodeMessage[ForceSwitchMessage](buffer)
	case MESSAGE_GAMEOVER:
		return decodeMessage[GameOverMessage](buffer)
	case MESSAGE_TURNRESOLVE:
		return decodeMessage[TurnResolvedMessage](buffer)
	case MESSAGE_SENDACTION:
		action, err := acceptAction(buffer)
		return SendActionMessage{Action: action}, err
	case MESSAGE_UPDATETIMER:
		return decodeMessage[UpdateTimerMessage](buffer)
	}

	return nil, &InvalidMsgTypeError{msgType: msgType}
}

func decodeMessage[T any](reader io.Reader) (T, error) {
	m := new(T)

	decoder := gob.NewDecoder(reader)
	err := decoder.Decode(m)
	return *m, err
}

func readAll(conn net.Conn) ([]byte, error) {
	finalBuf := make([]byte, 0)
	tempBufLen := 2048

	for {
		tempBuf := make([]byte, tempBufLen)

		n, err := conn.Read(tempBuf)
		// We finished reading
		if n < tempBufLen || err == io.EOF {
			finalBuf = append(finalBuf, tempBuf[:n]...)
			return finalBuf, nil
		}

		if err != nil {
			return nil, err
		}

		finalBuf = append(finalBuf, tempBuf[:n]...)
	}
}
