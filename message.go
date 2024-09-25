package main

import (
	"encoding/json"
	"net"
)

const (
	MT_CREATE_LOBBY = iota + 1
	MT_DESTROY_LOBBY
	MT_JOIN_LOBBY
)

type connMessage struct {
	conn        net.Conn
	messageType int
	message     string
}

func parseMessage(data string) (connMessage, error) {
	var parsedMessage connMessage
	err := json.Unmarshal([]byte(data), &parsedMessage)

	if err != nil {
		return parsedMessage, err
	}

	return parsedMessage, nil
}
