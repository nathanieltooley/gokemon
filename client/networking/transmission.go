package networking

import (
	"encoding/gob"
	"net"
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
