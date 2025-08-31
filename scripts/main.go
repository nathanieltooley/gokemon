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
		abilityMain("./poketerm/data/abilities.json")
	case "item":
		itemMain("./poketerm/data/items.json")
	case "move":
		moveMain("./poketerm/data/moves.json", "./poketerm/data/movesMap.json")
	}
}
