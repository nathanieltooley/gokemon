package state

import (
	"math"
	"slices"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
)

func TestSnapClean(t *testing.T) {
	snaps := []stateCore.StateSnapshot{
		{
			State: NewState(make([]core.Pokemon, 0), make([]core.Pokemon, 0)),
		},
		stateCore.NewEmptyStateSnapshot(),
		stateCore.NewMessageOnlySnapshot("Hello World!"),
	}

	newSnaps := cleanStateSnapshots(snaps)

	if len(newSnaps) != 1 {
		t.Fatalf("Incorrect snap size. Wanted 1, got %d. Snaps: %+v", len(newSnaps), newSnaps)
	}

	if !slices.Equal(newSnaps[0].Messages, []string{"Hello World!"}) {
		t.Fatalf("Incorrect state messages. Got %+v", newSnaps[0].Messages)
	}
}

func TestSandstormChip(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = core.WEATHER_SANDSTORM

	_ = ProcessTurn(&gameState, []stateCore.Action{})

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	damage := float64(pokemon.MaxHp) * (1.0 / 16.0)
	expectedHp := pokemon.MaxHp - uint(math.Ceil(damage))

	if pokemon.Hp.Value != expectedHp {
		t.Fatalf("pokemon hp did not match expected value. pokemon hp: %d/%d | expected: %d/%d", pokemon.Hp.Value, pokemon.MaxHp, expectedHp, pokemon.MaxHp)
	}
}

func getDummyPokemon() core.Pokemon {
	return game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
}

func getDummyPokemonWithAbility(ability string) core.Pokemon {
	pkm := getDummyPokemon()
	pkm.Ability.Name = ability

	return pkm
}

func getSimpleState(playerPkm core.Pokemon, enemyPkm core.Pokemon) stateCore.GameState {
	gameState := NewState([]core.Pokemon{playerPkm}, []core.Pokemon{enemyPkm})
	return gameState
}
