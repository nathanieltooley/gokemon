package main

import (
	"log"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) < 1 {
		log.Fatal("Not enough arguments")
	}

	scriptName := args[0]

	switch scriptName {
	case "ability":
		abilityMain()
	case "item":
		itemMain()
	case "move":
		moveMain()
	}
}
