package global

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/errorutils"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

type GlobalConfig struct {
	TeamSaveLocation string
	LocalPlayerName  string
	Debug            bool
}

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

	DownTabKey = key.NewBinding(key.WithKeys(tea.KeyTab.String()))
	UpTabKey   = key.NewBinding(key.WithKeys(tea.KeyShiftTab.String()))

	BackKey = key.NewBinding(key.WithKeys(tea.KeyEsc.String()))

	Opt = GlobalConfig{
		TeamSaveLocation: "",
		LocalPlayerName:  "Player",
	}

	GameTicksPerSecond = 20

	initLogger zerolog.Logger
)

func GlobalInit(files fs.FS, shouldLog bool) {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	configDir := DefaultConfigDir()
	configFilepath := DefaultConfigLocation()

	// Basic logging for config debugging
	initLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	// Initialize
	if err := os.MkdirAll(configDir, 0750); err != nil {
		initLogger.Err(err).Msg("error occured trying to create config dir")
	}

	// Read config file
	configContents, err := os.ReadFile(configFilepath)
	if err != nil {
		_, err := os.Create(configFilepath)
		if err != nil {
			initLogger.Err(err).Msg("error occurred while trying to create config file")
		}
	}

	// Non-empty config file
	if len(configContents) > 0 {
		newOpts := GlobalConfig{}
		if err := json.Unmarshal(configContents, &newOpts); err != nil {
			initLogger.Err(err).Msg("error occurred while trying to create new config file")
		} else {
			Opt = populateConfig(newOpts)
		}
	} else {
		config := populateConfig(GlobalConfig{})
		configBytes, err := json.Marshal(config)
		if err != nil {
			initLogger.Err(err).Msg("error occurred while trying to marshal default config values")
		}

		if err := os.WriteFile(configFilepath, configBytes, 0666); err != nil {
			initLogger.Err(err).Msg("error occurred while trying to write default config values")
		}

		Opt = config
	}

	level := zerolog.InfoLevel
	if Opt.Debug {
		level = zerolog.DebugLevel
	}

	// Logging Setup
	rollingWriter := NewRollingFileWriter(filepath.Join(configDir, "logs/"), "gokemon")
	// TODO: Make custom formatter. ConsoleWriter ends up printing out console format codes (obviously) that look bad in a text editor
	fileWriter := zerolog.ConsoleWriter{Out: rollingWriter}
	// Only used for init logging
	multiLogger := zerolog.New(zerolog.MultiLevelWriter(consoleWriter, fileWriter)).With().Timestamp().Logger().Level(level)

	initLogger = multiLogger
	if !shouldLog {
		initLogger = zerolog.Nop()
	}

	// Main global logger
	log.Logger = zerolog.New(fileWriter).With().Timestamp().Caller().Logger().Level(level)

	// Load concurrently
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		gen1Pokemon := loadPokemon(files, "data/gen1-data.csv")
		gen2Pokemon := loadPokemon(files, "data/gen2-data.csv")
		gen3Pokemon := loadPokemon(files, "data/gen3-data.csv")

		POKEMON = slices.Concat(gen1Pokemon, gen2Pokemon, gen3Pokemon)
		wg.Done()
	}()
	go func() {
		MOVES = loadMoves(files)
		wg.Done()
	}()
	go func() {
		ABILITIES = loadAbilities(files)
		wg.Done()
	}()
	go func() {
		ITEMS = loadItems(files)
		wg.Done()
	}()

	wg.Wait()
}

func loadPokemon(files fs.FS, filePath string) []game.BasePokemon {
	fileReader, err := files.Open(filePath)
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
		pokedexNumber := uint(errorutils.Must(strconv.ParseInt(row[0], 10, 16)))
		hp := uint(errorutils.Must(strconv.ParseInt(row[4], 10, 16)))
		attack := uint(errorutils.Must(strconv.ParseInt(row[5], 10, 16)))
		def := uint(errorutils.Must(strconv.ParseInt(row[6], 10, 16)))
		spAttack := uint(errorutils.Must(strconv.ParseInt(row[7], 10, 16)))
		spDef := uint(errorutils.Must(strconv.ParseInt(row[8], 10, 16)))
		speed := uint(errorutils.Must(strconv.ParseInt(row[9], 10, 16)))

		name := row[1]
		type1Name := row[2]
		type2Name := row[3]

		type1 := game.TYPE_MAP[type1Name]
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

	return pokemonList
}

func loadMoves(files fs.FS) *MoveRegistry {
	initLogger.Info().Msg("Loading Move Data")

	moveRegistry := new(MoveRegistry)
	movesPath := "data/moves.json"
	movesMapPath := "data/movesMap.json"

	moveDataFile, err := files.Open(movesPath)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read move data file")
	}
	defer moveDataFile.Close()

	moveDataBytes, err := io.ReadAll(moveDataFile)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read move data file")
	}

	moveMapFile, err := files.Open(movesMapPath)
	if err != nil {
		initLogger.Fatal().Err(err).Msg("Couldn't read move map file")
	}
	defer moveMapFile.Close()

	moveMapBytes, err := io.ReadAll(moveMapFile)

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

func loadAbilities(files fs.FS) map[string][]game.Ability {
	abilityFile := "data/abilities.json"
	file, err := files.Open(abilityFile)
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

func loadItems(files fs.FS) []string {
	itemsFile := "data/items.json"
	file, err := files.Open(itemsFile)
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

func populateConfig(config GlobalConfig) GlobalConfig {
	configDir := DefaultConfigDir()

	if config.LocalPlayerName == "" {
		config.LocalPlayerName = "Player"
	}
	if config.TeamSaveLocation == "" {
		config.TeamSaveLocation = filepath.Join(configDir, "saves/", "teams.json")
	}

	return config
}
