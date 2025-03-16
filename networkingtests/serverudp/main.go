package main

import (
	"net"
	"time"
)

func main() {
	pc, err := net.ListenPacket("udp4", ":7777")
	if err != nil {
		panic(err)
	}

	// 11000100.10101000.00000100.00000000
	// 11111111.11111111.11111100.00000000
	// 11000100.10101000.00000111.11111111
	// 192.168.7.255
	broadcastAddr, err := net.ResolveUDPAddr("udp4", "192.168.7.255:7777")
	if err != nil {
		panic(err)
	}

	for {
		_, err = pc.WriteTo([]byte("Test Broadcast!!!"), broadcastAddr)
		if err != nil {
			panic(err)
		}

		time.Sleep(1 * time.Second)
	}
}
