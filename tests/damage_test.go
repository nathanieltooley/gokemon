package tests

import (
	"math"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog/log"

	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
)

const iterCount = 1000

func TestDamage(t *testing.T) {
	global.StopLogging()
	defer global.ContinueLogging()
	for range iterCount {
		pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()
		enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()

		pokemon.Moves[0] = *global.MOVES.GetMove("tackle")

		damage := stateCore.Damage(pokemon, enemyPokemon, pokemon.Moves[0], false, core.WEATHER_NONE)

		checkDamageRange(t, damage, 29, 35)
	}
}

func TestDamageLow(t *testing.T) {
	log.Debug().Msg("====damage low test====")
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()

	global.ForceRng(&global.LowSource{})
	defer global.SetNormalRng()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	if damage != 29 {
		t.Fatalf("low damage incorrect: expected 29, got %d", damage)
	}
}

func TestDamageHigh(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()

	global.ForceRng(&global.HighSource{})
	defer global.SetNormalRng()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	if damage != 35 {
		t.Fatalf("high damage incorrect: expected 35, got %d", damage)
	}
}

func TestCritDamage(t *testing.T) {
	global.StopLogging()
	defer global.ContinueLogging()

	for range iterCount {
		pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()
		enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).SetPerfectIvs().SetLevel(100).Build()

		pokemon.Moves[0] = *global.MOVES.GetMove("tackle")

		damage := stateCore.Damage(pokemon, enemyPokemon, pokemon.Moves[0], true, core.WEATHER_NONE)

		checkDamageRange(t, damage, 44, 52)
	}
}

func TestSandstormChip(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = core.WEATHER_SANDSTORM

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	damage := float64(pokemon.MaxHp) * (1.0 / 16.0)
	expectedHp := pokemon.MaxHp - uint(math.Ceil(damage))

	if pokemon.Hp.Value != expectedHp {
		t.Fatalf("pokemon hp did not match expected value. pokemon hp: %d/%d | expected: %d/%d", pokemon.Hp.Value, pokemon.MaxHp, expectedHp, pokemon.MaxHp)
	}
}

func checkDamageRange(t *testing.T, damage uint, low uint, high uint) {
	if damage < low || damage > high {
		t.Fatalf("outside damage range: should be between %d - %d, got %d", low, high, damage)
	}
}
