package global

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/nathanieltooley/gokemon/client/errors"
	"github.com/nathanieltooley/gokemon/client/game"
	"golang.org/x/term"
)

var (
	TERM_WIDTH, TERM_HEIGHT, _ = term.GetSize(int(os.Stdout.Fd()))

	POKEMON   = loadPokemon()
	MOVES     = loadMoves()
	ABILITIES = loadAbilities()
	ITEMS     = loadItems()
)

func loadPokemon() game.PokemonRegistry {
	dataFile := "./data/gen1-data.csv"
	fileReader, err := os.Open(dataFile)
	defer fileReader.Close()

	if err != nil {
		log.Fatalln("Couldn't open Pokemon data file: ", err)
	}

	csvReader := csv.NewReader(fileReader)
	csvReader.Read()
	rows, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalln("Couldn't read Pokemon data file: ", err)
	}

	pokemonList := make([]game.BasePokemon, 0, len(rows))

	log.Println("Loading Pokemon Data")

	// Load CSV data
	for _, row := range rows {
		// These are "unwraped" because the data inserted should always follow this format
		pokedexNumber := int16(errors.Must(strconv.ParseInt(row[0], 10, 16)))
		hp := int16(errors.Must(strconv.ParseInt(row[4], 10, 16)))
		attack := int16(errors.Must(strconv.ParseInt(row[5], 10, 16)))
		def := int16(errors.Must(strconv.ParseInt(row[6], 10, 16)))
		spAttack := int16(errors.Must(strconv.ParseInt(row[7], 10, 16)))
		spDef := int16(errors.Must(strconv.ParseInt(row[8], 10, 16)))
		speed := int16(errors.Must(strconv.ParseInt(row[9], 10, 16)))

		name := row[1]
		type1Name := row[2]
		type2Name := row[3]

		var type1 *game.PokemonType = game.TYPE_MAP[type1Name]
		var type2 *game.PokemonType = nil

		if type2Name != "" {
			type2 = game.TYPE_MAP[type2Name]
		}

		newPokemon := game.BasePokemon{
			PokedexNumber: pokedexNumber,
			Name:          name,
			Type1:         type1,
			Type2:         type2,
			Hp:            hp,
			Attack:        attack,
			Def:           def,
			SpAttack:      spAttack,
			SpDef:         spDef,
			Speed:         speed,
		}

		pokemonList = append(pokemonList, newPokemon)
	}

	log.Printf("Loaded %d pokemon\n", len(pokemonList))

	if err != nil {
		log.Fatalf("Failed to load pokemon data: %s\n", err)
	}

	return game.PokemonRegistry(pokemonList)
}

func loadMoves() *game.MoveRegistry {
	log.Println("Loading Move Data")

	moveRegistry := new(game.MoveRegistry)
	movesPath := "./data/moves.json"
	movesMapPath := "./data/movesMap.json"

	moveDataBytes, err := os.ReadFile(movesPath)
	if err != nil {
		log.Fatalln("Couldn't read move data file: ", err)
	}

	moveMapBytes, err := os.ReadFile(movesMapPath)
	if err != nil {
		log.Fatalln("Couldn't read move map file: ", err)
	}

	parsedMoves := make([]game.MoveFull, 0, 1000)
	moveMap := make(map[string][]string)

	if err := json.Unmarshal(moveDataBytes, &parsedMoves); err != nil {
		log.Fatalln("Couldn't unmarshal move data: ", err)
	}
	if err := json.Unmarshal(moveMapBytes, &moveMap); err != nil {
		log.Fatalln("Couldn't unmarshal move map: ", err)
	}

	moveRegistry.MoveList = parsedMoves
	moveRegistry.MoveMap = moveMap

	if err != nil {
		log.Fatalf("Failed to load move data: %s\n", err)
	}

	log.Printf("Loaded %d moves\n", len(moveRegistry.MoveList))
	log.Printf("Loaded move info for %d pokemon\n", len(moveRegistry.MoveMap))

	return moveRegistry
}

func loadAbilities() map[string][]string {
	abilityFile := "./data/abilities.json"
	file, err := os.Open(abilityFile)
	if err != nil {
		log.Fatalln("Couldn't open abilities file: ", err)
	}

	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Fatalln("Couldn't read abilities file: ", err)
	}

	abilityMap := make(map[string][]string)
	if err := json.Unmarshal(fileData, &abilityMap); err != nil {
		log.Fatalln("Couldn't unmarshal ability data: ", err)
	}

	log.Printf("Loaded abilities for %d pokemon\n", len(abilityMap))
	return abilityMap
}

func loadItems() []string {
	itemsFile := "./data/items.json"
	file, err := os.Open(itemsFile)
	if err != nil {
		log.Fatalln("Couldn't open items file: ", err)
	}

	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Fatalln("Couldn't read items file: ", err)
	}
	items := make([]string, 0)
	if err := json.Unmarshal(fileData, &items); err != nil {
		log.Fatalln("Couldn't parse items.json: ", err)
	}

	log.Printf("Loaded %d items\n", len(items))
	return items
}
