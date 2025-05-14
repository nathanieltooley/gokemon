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

func TestDrizzle(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("bulbasaur")).Build()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build()

	pokemon.Ability.Name = "drizzle"

	gameState := state.NewState([]game.Pokemon{pokemon}, []game.Pokemon{enemyPokemon})
	_ = stateupdater.ProcessTurn(&gameState, []state.Action{state.NewSwitchAction(&gameState, state.HOST, 0)})

	if gameState.Weather != game.WEATHER_RAIN {
		t.Fatalf("Expected weather to be rain: got %d", gameState.Weather)
	}
}
