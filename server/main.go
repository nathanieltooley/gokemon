package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

const PORT uint16 = 8080
const MAX_READ_SIZE uint16 = 2048

func handleConnReads(conn net.Conn, lobbyCreate chan connMessage, lobbyDestroy chan connMessage) {
	defer func() {
		log.Println("Closing connection")
		conn.Close()
	}()

	addr := conn.RemoteAddr()
	readBuffer := make([]byte, 2048)

	for {
		dataLength, err := conn.Read(readBuffer)

		if err != nil {
			log.Printf("Error from %s, %s\n", addr, err)
			return
		}

		var messageBuilder strings.Builder
		for i := 0; i < dataLength; i++ {
			messageBuilder.WriteByte(readBuffer[i])
		}
		message := messageBuilder.String()

		log.Printf("Message from connection %s: %s", addr, message)
		parsedMessage, err := parseMessage(message)

		if err != nil {
			log.Printf("Connection, %s, sent an invalid message: %s\n", conn.RemoteAddr(), err)
		}

		switch parsedMessage.messageType {
		case MT_CREATE_LOBBY:
			lobbyCreate <- parsedMessage
		case MT_DESTROY_LOBBY:
			lobbyDestroy <- parsedMessage
		}
	}
}

func createLobbyHandler(lobbyCreate chan connMessage, openLobbies *[]net.Conn) {
	for {
		msg := <-lobbyCreate

		*openLobbies = append(*openLobbies, msg.conn)
		log.Printf("Created lobby: %s\n", msg.conn.RemoteAddr())
	}
}

// this takes in a pointer to a slice so that the slice's pointer is actually changed
func destroyLobbyHandler(lobbyDestroy chan connMessage, openLobbies *[]net.Conn) {
	for {
		// find the lobby to be removed
		msg := <-lobbyDestroy
		removeIndex := -1
		for i, conn := range *openLobbies {
			if conn.RemoteAddr() == msg.conn.RemoteAddr() {
				removeIndex = i
			}
		}

		if removeIndex != -1 {
			// remove the closed lobby
			newLobbies := append((*openLobbies)[0:removeIndex], (*openLobbies)[removeIndex+1:]...)
			// set the pointer to the new slice
			*openLobbies = newLobbies
			log.Printf("Destroyed lobby: %s\n", msg.conn.RemoteAddr())
		}
	}
}

func main() {
	openLobbies := make([]net.Conn, 0)

	lobbyCreationChan := make(chan connMessage)
	lobbyDestructionChan := make(chan connMessage)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))

	if err != nil {
		log.Fatalf("Server could not listen on port %d: %s\n", PORT, err)
	}

	log.Printf("Server is now listening on port %d\n", PORT)
	go createLobbyHandler(lobbyCreationChan, &openLobbies)
	go destroyLobbyHandler(lobbyDestructionChan, &openLobbies)

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("Error while trying to accept connection: %s\n", err)
		}

		log.Printf("Accepting connection from %s\n", conn.RemoteAddr())
		go handleConnReads(conn, lobbyCreationChan, lobbyDestructionChan)
	}
}
