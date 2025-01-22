package global

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/nathanieltooley/gokemon/client/errors"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

var (
	TERM_WIDTH, TERM_HEIGHT, _ = term.GetSize(int(os.Stdout.Fd()))

	POKEMON   PokemonRegistry
	MOVES     *MoveRegistry
	ABILITIES map[string][]game.Ability
	ITEMS     []string

	SelectKey = key.NewBinding(
		key.WithKeys("enter"),
	)
	MoveLeftKey = key.NewBinding(
		key.WithKeys("left", "h"),
	)
	MoveRightKey = key.NewBinding(
		key.WithKeys("right", "l"),
	)
	MoveDownKey = key.NewBinding(
		key.WithKeys("down", "j"),
	)
	MoveUpKey = key.NewBinding(
		key.WithKeys("up", "k"),
	)

	// TeamSaveLocation = "./saves/teams.json"
	TeamSaveLocation = "./saves"

	initLogger zerolog.Logger
)

func GlobalInit() {
	rollingWriter := rollingFileWriter{}
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	// TODO: Custom formatter, ends up printing out console format codes (obviously)
	fileWriter := zerolog.ConsoleWriter{Out: rollingWriter}
	multiLogger := zerolog.New(zerolog.MultiLevelWriter(consoleWriter, fileWriter)).With().Timestamp().Logger()

	initLogger = multiLogger
	log.Logger = zerolog.New(fileWriter).With().Timestamp().Caller().Logger()

	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		POKEMON = loadPokemon()
		wg.Done()
	}()
	go func() {
		MOVES = loadMoves()
		wg.Done()
	}()
	go func() {
		ABILITIES = loadAbilities()
		wg.Done()
	}()
	go func() {
		ITEMS = loadItems()
		wg.Done()
	}()

	wg.Wait()
}

func loadPokemon() PokemonRegistry {
	dataFile := "./data/gen1-data.csv"
	fileReader, err := os.Open(dataFile)

	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't open Pokemon data file")
	}

	defer fileReader.Close()

	csvReader := csv.NewReader(fileReader)
	csvReader.Read()
	rows, err := csvReader.ReadAll()
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read Pokemon data file")
	}

	pokemonList := make([]game.BasePokemon, 0, len(rows))

	initLogger.Info().Msg("Loading Pokemon Data")

	// Load CSV data
	for _, row := range rows {
		// These are "unwraped" because the data inserted should always follow this format
		pokedexNumber := uint(errors.Must(strconv.ParseInt(row[0], 10, 16)))
		hp := uint(errors.Must(strconv.ParseInt(row[4], 10, 16)))
		attack := uint(errors.Must(strconv.ParseInt(row[5], 10, 16)))
		def := uint(errors.Must(strconv.ParseInt(row[6], 10, 16)))
		spAttack := uint(errors.Must(strconv.ParseInt(row[7], 10, 16)))
		spDef := uint(errors.Must(strconv.ParseInt(row[8], 10, 16)))
		speed := uint(errors.Must(strconv.ParseInt(row[9], 10, 16)))

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

	initLogger.Info().Msgf("Loaded %d pokemon", len(pokemonList))

	if err != nil {
		initLogger.Fatal().Err(err).Msg("Failed to load pokemon data")
	}

	return PokemonRegistry(pokemonList)
}

func loadMoves() *MoveRegistry {
	initLogger.Info().Msg("Loading Move Data")

	moveRegistry := new(MoveRegistry)
	movesPath := "./data/moves.json"
	movesMapPath := "./data/movesMap.json"

	moveDataBytes, err := os.ReadFile(movesPath)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read move data file")
	}

	moveMapBytes, err := os.ReadFile(movesMapPath)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read move map file")
	}

	parsedMoves := make([]game.Move, 0, 1000)
	moveMap := make(map[string][]string)

	if err := json.Unmarshal(moveDataBytes, &parsedMoves); err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't unmarshal move data")
	}
	if err := json.Unmarshal(moveMapBytes, &moveMap); err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't unmarshal move map")
	}

	moveRegistry.MoveList = parsedMoves
	moveRegistry.MoveMap = moveMap

	if err != nil {
		initLogger.Fatal().Err(err).Msg("Failed to load move data")
	}

	initLogger.Info().Msgf("Loaded %d moves", len(moveRegistry.MoveList))
	initLogger.Info().Msgf("Loaded move info for %d pokemon", len(moveRegistry.MoveMap))

	return moveRegistry
}

func loadAbilities() map[string][]game.Ability {
	abilityFile := "./data/abilities.json"
	file, err := os.Open(abilityFile)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't open abilities file")
	}

	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read abilities file")
	}

	abilityMap := make(map[string][]game.Ability)
	if err := json.Unmarshal(fileData, &abilityMap); err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't unmarshal ability data")
	}

	initLogger.Info().Msgf("Loaded abilities for %d pokemon", len(abilityMap))
	return abilityMap
}

func loadItems() []string {
	itemsFile := "./data/items.json"
	file, err := os.Open(itemsFile)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't open items file")
	}

	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read items file")
	}
	items := make([]string, 0)
	if err := json.Unmarshal(fileData, &items); err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't parse items.json")
	}

	initLogger.Info().Msgf("Loaded %d items", len(items))
	return items
}

type MoveRegistry struct {
	// TODO: Maybe make this a map
	MoveList []game.Move
	MoveMap  map[string][]string
}

func (m *MoveRegistry) GetMove(name string) *game.Move {
	for _, move := range m.MoveList {
		if move.Name == name {
			return &move
		}
	}

	return nil
}

func (m *MoveRegistry) GetFullMovesForPokemon(pokemonName string) []*game.Move {
	pokemonLowerName := strings.ToLower(pokemonName)
	moves := m.MoveMap[pokemonLowerName]
	movesFull := make([]*game.Move, 0, len(moves))

	for _, moveName := range moves {
		movesFull = append(movesFull, m.GetMove(moveName))
	}

	return movesFull
}

type PokemonRegistry []game.BasePokemon

func (p PokemonRegistry) GetPokemonByPokedex(pkdNumber int) *game.BasePokemon {
	for _, pkm := range p {
		if pkm.PokedexNumber == uint(pkdNumber) {
			return &pkm
		}
	}

	return nil
}

func (p PokemonRegistry) GetPokemonByName(pkmName string) *game.BasePokemon {
	for _, pkm := range p {
		if strings.EqualFold(pkm.Name, pkmName) {
			return &pkm
		}
	}

	return nil
}

func (p PokemonRegistry) GetRandomPokemon() *game.BasePokemon {
	pkmIndex := rand.IntN(len(p))

	return &p[pkmIndex]
}

func GetAbilitiesForPokemon(name string) []game.Ability {
	pokemonLowerName := strings.ToLower(name)

	return ABILITIES[pokemonLowerName]
}
