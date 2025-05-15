package tests

import (
	"os"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/game/state/stateupdater"
	"github.com/nathanieltooley/gokemon/client/global"
)

func init() {
	global.GlobalInit(os.DirFS("../"), false)
}

// NOTE: Most ability tests will directly set the ability on a pokemon (probably bulbasaur) rather than choosing a pokemon
// with that ability for the simple fact that it really doesn't matter. However it may change if for some reason it's important to the ability
func TestDrizzle(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).Build()
	enemyPokemon := getDummyPokemon()

	pokemon.Ability.Name = "drizzle"

	gameState := state.NewState([]game.Pokemon{pokemon}, []game.Pokemon{enemyPokemon})
	_ = stateupdater.ProcessTurn(&gameState, []state.Action{state.NewSwitchAction(&gameState, state.HOST, 0)})

	if gameState.Weather != game.WEATHER_RAIN {
		t.Fatalf("Expected weather to be rain: got %d", gameState.Weather)
	}
}

func TestSpeedBoost(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
	enemyPokemon := getDummyPokemon()
	pokemon.Ability.Name = "speed-boost"

	gameState := state.NewState([]game.Pokemon{pokemon}, []game.Pokemon{enemyPokemon})

	if gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage != 0 {
		t.Fatal("test pokemon has started with incorrect speed stage")
	}

	_ = stateupdater.ProcessTurn(&gameState, []state.Action{})

	pokemonSpeedStage := gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage is incorrect: got %d expected 1", pokemonSpeedStage)
	}

	enemyPokemonSpeedStage := gameState.ClientPlayer.GetActivePokemon().RawSpeed.Stage
	if enemyPokemonSpeedStage != 0 {
		t.Fatal("enemy pokemon's speed stage boosted incorrectly to 1")
	}

	// Test that pokemon that switch in do not get the speed boost
	_ = stateupdater.ProcessTurn(&gameState, []state.Action{state.NewSwitchAction(&gameState, state.HOST, 0)})

	pokemonSpeedStage = gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage should have stayed at 1: got %d instead", pokemonSpeedStage)
	}
}

func getDummyPokemon() game.Pokemon {
	return game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build()
}
