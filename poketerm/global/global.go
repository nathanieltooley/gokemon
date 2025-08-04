package global

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-logr/zerologr"
	"github.com/nathanieltooley/gokemon/golurk"
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

	// Global RNG for unimportant or non-networked RNG
	GokeRand = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	initLogger    zerolog.Logger
	previousLevel zerolog.Level
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

	multiLogger := zerolog.New(zerolog.MultiLevelWriter(consoleWriter, createFileWriter(configDir))).With().Timestamp().Logger().Level(level)

	initLogger = multiLogger
	if !shouldLog {
		initLogger = zerolog.Nop()
	}

	// Main global logger
	log.Logger = createLogger(configDir, level)
	golurk.SetInternalLogger(zerologr.New(&log.Logger))

	errs := golurk.DefaultLoader(files)
	if len(errs) > 0 {
		for _, err := range errs {
			log.Err(err)
			fmt.Printf("err: %s", err)
		}

		panic("error(s) encountered while loading pokemon data")
	}
}

func StopLogging() {
	previousLevel = log.Logger.GetLevel()
	log.Logger = zerolog.Nop()
}

func ContinueLogging() {
	log.Logger = createLogger(DefaultConfigDir(), previousLevel)
}

func UpdateLogLevel(level zerolog.Level) {
	log.Logger = log.Logger.Level(level)
}

func ForceRng(source rand.Source) {
	GokeRand = rand.New(source)
}

func SetNormalRng() {
	GokeRand = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
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

func createFileWriter(configDir string) zerolog.ConsoleWriter {
	rollingWriter := NewRollingFileWriter(filepath.Join(configDir, "logs/"), "gokemon")
	// TODO: Make custom formatter. ConsoleWriter ends up printing out console format codes (obviously) that look bad in a text editor
	return zerolog.ConsoleWriter{Out: rollingWriter}
}

func createLogger(configDir string, level zerolog.Level) zerolog.Logger {
	// Main global logger
	return zerolog.New(createFileWriter(configDir)).With().Timestamp().Caller().Logger().Level(level)
}
