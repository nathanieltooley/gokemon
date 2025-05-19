package tests

import (
	"os"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/game/state"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
)

func init() {
	global.GlobalInit(os.DirFS("../"), false)
	global.UpdateLogLevel(zerolog.DebugLevel)
}

// NOTE: Most ability tests will directly set the ability on a pokemon (probably bulbasaur) rather than choosing a pokemon
// with that ability for the simple fact that it really doesn't matter. However it may change if for some reason it's important to the ability
func TestDrizzle(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).Build()
	enemyPokemon := getDummyPokemon()

	pokemon.Ability.Name = "drizzle"

	gameState := state.NewState([]core.Pokemon{pokemon}, []core.Pokemon{enemyPokemon})
	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)})

	if gameState.Weather != core.WEATHER_RAIN {
		t.Fatalf("Expected weather to be rain: got %d", gameState.Weather)
	}
}

func TestSpeedBoost(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
	enemyPokemon := getDummyPokemon()
	pokemon.Ability.Name = "speed-boost"

	gameState := state.NewState([]core.Pokemon{pokemon}, []core.Pokemon{enemyPokemon})

	if gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage != 0 {
		t.Fatal("test pokemon has started with incorrect speed stage")
	}

	_ = state.ProcessTurn(&gameState, []stateCore.Action{})

	pokemonSpeedStage := gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage is incorrect: got %d expected 1", pokemonSpeedStage)
	}

	enemyPokemonSpeedStage := gameState.ClientPlayer.GetActivePokemon().RawSpeed.Stage
	if enemyPokemonSpeedStage != 0 {
		t.Fatal("enemy pokemon's speed stage boosted incorrectly to 1")
	}

	// Test that pokemon that switch in do not get the speed boost
	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)})

	pokemonSpeedStage = gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage should have stayed at 1: got %d instead", pokemonSpeedStage)
	}
}

func TestSturdy(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("charizard")).SetLevel(100).SetPerfectIvs().Build()

	pokemon.Ability.Name = "sturdy"
	enemyPokemon.Moves[0] = *global.MOVES.GetMove("ember")

	gameState := state.NewState([]core.Pokemon{pokemon}, []core.Pokemon{enemyPokemon})

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.AI, 0)})

	if gameState.HostPlayer.GetActivePokemon().Hp.Value != 1 {
		t.Fatalf("pokemon should survive with 1 hp because of sturdy, got %d", pokemon.Hp.Value)
	}
}

func TestDamp(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.Ability.Name = "damp"
	enemyPokemon.Moves[0] = *global.MOVES.GetMove("self-destruct")

	gameState := state.NewState([]core.Pokemon{pokemon}, []core.Pokemon{enemyPokemon})

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	if pokemon.Hp.Value != pokemon.MaxHp || enemyPokemon.Hp.Value != enemyPokemon.MaxHp {
		t.Fatalf("self-destruct most likely activated. player pokemon hp: %d/%d | enemy pokemon hp: %d|%d", pokemon.Hp.Value, pokemon.MaxHp, enemyPokemon.Hp.Value, enemyPokemon.MaxHp)
	}
}

func getDummyPokemon() core.Pokemon {
	return game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
}
