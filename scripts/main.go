package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	generationLimit := flag.Int("gen", 0, "Limits abilities to before and in the generation provided")
	scriptName := flag.String("script", "", "What script to run")

	flag.Parse()

	args := os.Args[1:]

	if len(args) < 1 {
		log.Fatal("Not enough arguments")
	}

	switch *scriptName {
	case "ability":
		abilityMain("./poketerm/data/abilities.json", *generationLimit)
	case "item":
		itemMain("./poketerm/data/items.json")
	case "move":
		moveMain("./poketerm/data/moves.json", "./poketerm/data/movesMap.json")
	}
}
