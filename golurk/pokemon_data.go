package golurk

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
)

var GlobalData = pokemonDb{}

type pokemonDb struct {
	moves     MoveRegistry
	Pokemon   []BasePokemon
	abilities AbilityRegistry
	Items     []string
}

type MoveRegistry struct {
	// Moves is a map of move names to full move info
	Moves map[string]Move
	// LearnedPokemonMoves is a map that turns Pokemon names into lists of move names
	LearnedPokemonMoves map[string][]string
}

type AbilityRegistry struct {
	// for right now this is unused
	Abilities        []Ability
	PokemonAbilities map[string][]Ability
}

func SetGlobalMoves(mr MoveRegistry) {
	GlobalData.moves = mr
}

func SetGlobalAbilities(ar AbilityRegistry) {
	GlobalData.abilities = ar
}

func (db pokemonDb) GetMove(name string) *Move {
	move, ok := db.moves.Moves[name]
	if ok {
		return &move
	} else {
		return nil
	}
}

func (db pokemonDb) GetFullMovesForPokemon(pokemonName string) []Move {
	pokemonLowerName := strings.ToLower(pokemonName)
	moves := db.moves.LearnedPokemonMoves[pokemonLowerName]
	movesFull := make([]Move, 0, len(moves))

	for _, moveName := range moves {
		optionalMove := db.GetMove(moveName)
		if optionalMove != nil {
			movesFull = append(movesFull, *optionalMove)
		}
	}

	return movesFull
}

func (db pokemonDb) GetPokemonByPokedex(pkdNumber int) *BasePokemon {
	for _, pkm := range db.Pokemon {
		if pkm.PokedexNumber == uint(pkdNumber) {
			return &pkm
		}
	}

	return nil
}

func (db pokemonDb) GetPokemonByName(pkmName string) *BasePokemon {
	for _, pkm := range db.Pokemon {
		if strings.EqualFold(pkm.Name, pkmName) {
			return &pkm
		}
	}

	return nil
}

func (db pokemonDb) GetRandomPokemon() BasePokemon {
	pkmIndex := rand.IntN(len(db.Pokemon))

	return db.Pokemon[pkmIndex]
}

func (db pokemonDb) GetPokemonAbilities(name string) []Ability {
	pokemonLowerName := strings.ToLower(name)

	return db.abilities.PokemonAbilities[pokemonLowerName]
}

// func (db pokemonDb) GetAbilities() []Ability {
// 	return db.abilities.Abilities
// }

// LoadPokemon takes in the bytes of a csv file with the following columns:
// PokedexNumber, HP, Attack, Defense, SpecialAttack, SpecialDefense, Speed
// in that order. All values must be valid integers
func LoadPokemon(fileBytes []byte) ([]BasePokemon, error) {
	// fileReader, err := files.Open(filePath)
	// if err != nil {
	// 	internalLogger.Error(err, "Couldn't open Pokemon data file")
	// }
	//
	// defer fileReader.Close()

	csvReader := csv.NewReader(bytes.NewBuffer(fileBytes))
	csvReader.Read()
	rows, err := csvReader.ReadAll()
	if err != nil {
		internalLogger.Error(err, "invalid csv data")
	}

	pokemonList := make([]BasePokemon, 0, len(rows))

	internalLogger.Info("Loading Pokemon Data")

	// Load CSV data
	for _, row := range rows {
		var err error
		var pokedexNumber int64
		var hp int64
		var attack int64
		var def int64
		var spAttack int64
		var spDef int64
		var speed int64

		pokedexNumber, err = strconv.ParseInt(row[0], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid pokedex number")
			return nil, err
		}
		hp, err = strconv.ParseInt(row[4], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid hp")
			return nil, err
		}
		attack, err = strconv.ParseInt(row[5], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid attack")
			return nil, err
		}
		def, err = strconv.ParseInt(row[6], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid defense")
			return nil, err
		}
		spAttack, err = strconv.ParseInt(row[7], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid special attack")
			return nil, err
		}
		spDef, err = strconv.ParseInt(row[8], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid special defense")
			return nil, err
		}
		speed, err = strconv.ParseInt(row[9], 10, 16)
		if err != nil {
			internalLogger.WithName("pokemon_parsing").Error(err, "invalid speed")
			return nil, err
		}

		name := row[1]
		type1Name := row[2]
		type2Name := row[3]

		internalLogger.WithName("load_pokemon").V(1).Info("loaded pokemon", "pokedex", pokedexNumber, "name", name, "hp", hp, "attack", attack, "def", def, "spattack", spAttack, "spDef", spDef, "speed", speed)

		type1 := TYPE_MAP[type1Name]
		var type2 *PokemonType = nil

		if type2Name != "" {
			type2 = TYPE_MAP[type2Name]
		}

		newPokemon := BasePokemon{
			PokedexNumber: uint(pokedexNumber),
			Name:          name,
			Type1:         type1,
			Type2:         type2,
			Hp:            uint(hp),
			Attack:        uint(attack),
			Def:           uint(def),
			SpAttack:      uint(spAttack),
			SpDef:         uint(spDef),
			Speed:         uint(speed),
		}

		pokemonList = append(pokemonList, newPokemon)
	}

	internalLogger.Info("Loaded pokemon", "count", len(pokemonList))

	return pokemonList, nil
}

// LoadMoves takes in json that lists out move information and json that maps pokemon names to what moves they can learn
func LoadMoves(moveBytes []byte, moveMapBytes []byte) (MoveRegistry, error) {
	internalLogger.Info("Loading Move Data")

	parsedMoves := make([]Move, 0, 1000)
	moveMap := make(map[string][]string)
	moveRegistry := MoveRegistry{Moves: make(map[string]Move)}

	if err := json.Unmarshal(moveBytes, &parsedMoves); err != nil {
		internalLogger.Error(err, "Couldn't unmarshal move data")
		return moveRegistry, err
	}
	if err := json.Unmarshal(moveMapBytes, &moveMap); err != nil {
		internalLogger.Error(err, "Couldn't unmarshal move map")
		return moveRegistry, err
	}

	// convert move slice to move name -> move map
	for _, parsedMove := range parsedMoves {
		moveRegistry.Moves[parsedMove.Name] = parsedMove
	}

	moveRegistry.LearnedPokemonMoves = moveMap

	internalLogger.Info("Loaded moves", "count", len(moveRegistry.Moves), "pokemon_count", len(moveRegistry.LearnedPokemonMoves))

	return moveRegistry, nil
}

// LoadAbilities takes in json that lists ability info and json that maps pokemon names to abilities
// (this is different from loadMoves because info about Moves is much larger than info about abiltiies)
func LoadAbilities(abilitiesMapBytes []byte) (AbilityRegistry, error) {
	abilitiesList := []Ability{}
	abilityMap := make(map[string][]Ability)

	if err := json.Unmarshal(abilitiesMapBytes, &abilityMap); err != nil {
		internalLogger.Error(err, "Invalid ability map")
		return AbilityRegistry{}, err
	}

	internalLogger.Info("Loaded abilities", "pokemon_count", len(abilityMap))
	// TODO: Actually load abilities list. Right now its fine because nothing uses it
	return AbilityRegistry{Abilities: abilitiesList, PokemonAbilities: abilityMap}, nil
}

func LoadItems(itemBytes []byte) ([]string, error) {
	items := make([]string, 0)
	if err := json.Unmarshal(itemBytes, &items); err != nil {
		internalLogger.Error(err, "Couldn't parse items.json")
		return items, err
	}

	internalLogger.Info("Loaded %d items", "count", len(items))
	return items, nil
}

// DefaultLoader loads pokemon data from their locations in the gokemon repo.
// This data is used in the terminal program and for tests in this library.
// Should NOT be used outside of those two contexts unless your files are formatted
// the exact same way.
func DefaultLoader(files fs.FS) []error {
	// Load concurrently
	var wg sync.WaitGroup
	wg.Add(4)
	errChan := make(chan error, 8)

	go func() {
		defer wg.Done()

		genCount := 3
		pokemon := make([]BasePokemon, 0, genCount*150)
		for i := range genCount {
			genPath := fmt.Sprintf("data/gen%d-data.csv", i+1)
			genFile, err := files.Open(genPath)
			if err != nil {
				errChan <- err
				return
			}

			genBytes, err := fileReadAll(genFile)
			if err != nil {
				errChan <- err
				return
			}

			genPokemon, err := LoadPokemon(genBytes)
			if err != nil {
				errChan <- err
				return
			}

			pokemon = append(pokemon, genPokemon...)
		}

		GlobalData.Pokemon = pokemon
	}()
	go func() {
		defer wg.Done()

		moveFile, err := files.Open("data/moves.json")
		if err != nil {
			errChan <- err
			return
		}

		moveMapFile, err := files.Open("data/movesMap.json")
		if err != nil {
			errChan <- err
			return
		}

		moveBytes, err := fileReadAll(moveFile)
		if err != nil {
			errChan <- err
			return
		}

		moveMapBytes, err := fileReadAll(moveMapFile)
		if err != nil {
			errChan <- err
			return
		}

		moves, err := LoadMoves(moveBytes, moveMapBytes)
		if err != nil {
			errChan <- err
			return
		}

		SetGlobalMoves(moves)
	}()
	go func() {
		defer wg.Done()

		abilityFile, err := files.Open("data/abilities.json")
		if err != nil {
			errChan <- err
			return
		}

		abilityBytes, err := fileReadAll(abilityFile)
		if err != nil {
			errChan <- err
			return
		}

		abilities, err := LoadAbilities(abilityBytes)
		if err != nil {
			errChan <- err
		}
		SetGlobalAbilities(abilities)
	}()
	go func() {
		defer wg.Done()

		itemFile, err := files.Open("data/items.json")
		if err != nil {
			errChan <- err
			return
		}

		itemBytes, err := fileReadAll(itemFile)
		if err != nil {
			errChan <- err
			return
		}

		items, err := LoadItems(itemBytes)
		if err != nil {
			errChan <- err
			return
		}

		GlobalData.Items = items
	}()

	wg.Wait()
	errs := make([]error, 0)
	for {
		shouldBreak := false
		select {
		case err := <-errChan:
			errs = append(errs, err)
		default:
			shouldBreak = true
		}

		if shouldBreak {
			break
		}
	}

	return errs
}

func fileReadAll(file fs.File) ([]byte, error) {
	var fileSize int64
	stat, err := file.Stat()
	if err == nil {
		fileSize = stat.Size()
	}
	buf := make([]byte, fileSize)

	len, err := file.Read(buf)
	return buf[0:len], err
}
