package networking

import (
	"encoding/gob"
	"net"
	"reflect"

	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
)

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
	interfaceName := reflect.TypeOf(data).String()

	_, err := conn.Write([]byte(interfaceName))

	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(conn)
	return encoder.Encode(data)
}

func AcceptAction(conn net.Conn) (state.Action, error) {
	interfaceNameBytes := make([]byte, 128)

	n, err := conn.Read(interfaceNameBytes)
	if err != nil {
		return nil, err
	}

	decoder := gob.NewDecoder(conn)

	interfaceName := string(interfaceNameBytes[0:n])
	log.Info().Msgf("host got action type: %s", interfaceName)

	switch interfaceName {
	case "*state.SwitchAction":
		a := &state.SwitchAction{}

		err := decoder.Decode(a)
		return a, err
	case "*state.SkipAction":
		a := &state.SkipAction{}

		err := decoder.Decode(a)
		return a, err
	case "*state.AttackAction":
		a := &state.AttackAction{}

		err := decoder.Decode(a)
		return a, err
	}

	log.Fatal().Msg("tried to accept invalid action")

	return nil, nil
}
