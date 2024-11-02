package global

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/nathanieltooley/gokemon/client/errors"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/reg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

var (
	TERM_WIDTH, TERM_HEIGHT, _ = term.GetSize(int(os.Stdout.Fd()))

	POKEMON   reg.PokemonRegistry
	MOVES     *reg.MoveRegistry
	ABILITIES map[string][]string
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

	initLogger zerolog.Logger
)

func init() {
	logFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	if err != nil {
		fmt.Println("Could not open file 'client.log' for logging. logging will be console only")
		initLogger = zerolog.New(consoleWriter).With().Timestamp().Logger()
	} else {
		// TODO: Custom formatter, ends up printing out console format codes (obviously)
		fileWriter := zerolog.ConsoleWriter{Out: logFile}
		multiLogger := zerolog.New(zerolog.MultiLevelWriter(consoleWriter, fileWriter)).With().Timestamp().Logger()

		initLogger = multiLogger
		log.Logger = zerolog.New(fileWriter).With().Timestamp().Logger()
	}

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

func loadPokemon() reg.PokemonRegistry {
	dataFile := "./data/gen1-data.csv"
	fileReader, err := os.Open(dataFile)
	defer fileReader.Close()

	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't open Pokemon data file")
	}

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

	initLogger.Info().Msgf("Loaded %d pokemon", len(pokemonList))

	if err != nil {
		initLogger.Fatal().Err(err).Msg("Failed to load pokemon data")
	}

	return reg.PokemonRegistry(pokemonList)
}

func loadMoves() *reg.MoveRegistry {
	initLogger.Info().Msg("Loading Move Data")

	moveRegistry := new(reg.MoveRegistry)
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

	parsedMoves := make([]game.MoveFull, 0, 1000)
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

func loadAbilities() map[string][]string {
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

	abilityMap := make(map[string][]string)
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
