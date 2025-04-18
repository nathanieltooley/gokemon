package global

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
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

func GlobalInit(files fs.FS) {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	configDir := DefaultConfigDir()
	configFileName := DefaultConfigLocation()

	// Logging Setup
	rollingWriter := NewRollingFileWriter(filepath.Join(configDir, "logs/"), "gokemon")
	// TODO: Custom formatter, ends up printing out console format codes (obviously)
	fileWriter := zerolog.ConsoleWriter{Out: rollingWriter}
	multiLogger := zerolog.New(zerolog.MultiLevelWriter(consoleWriter, fileWriter)).With().Timestamp().Logger()

	initLogger = multiLogger
	log.Logger = zerolog.New(fileWriter).With().Timestamp().Caller().Logger()

	if err := os.MkdirAll(configDir, 0750); err != nil {
		initLogger.Err(err).Msg("error occured trying to create config dir")
	}

	// Read config file
	configFile, err := os.ReadFile(configFileName)
	if err != nil {
		_, err := os.Create(configFileName)
		if err != nil {
			initLogger.Err(err).Msg("error occurred while trying to create config file")
		}
	}

	if len(configFile) > 0 {
		newOpts := GlobalConfig{}
		if err := json.Unmarshal(configFile, &newOpts); err != nil {
			initLogger.Err(err).Msg("error occurred while trying to create new config file")
		} else {
			Opt = newOpts
		}
	} else {
		// default team save location
		Opt.TeamSaveLocation = filepath.Join(configDir, "saves/", "teams.json")
	}

	// Load concurrently
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		POKEMON = loadPokemon(files)
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

func loadPokemon(files fs.FS) PokemonRegistry {
	dataFile := "data/gen1-data.csv"
	fileReader, err := files.Open(dataFile)
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

	return PokemonRegistry(pokemonList)
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
