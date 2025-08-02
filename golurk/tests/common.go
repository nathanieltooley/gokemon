// Package tests contains integration tests between different packages
package tests

import (
	"math"

	"github.com/nathanieltooley/gokemon/golurk"
)

func getDummyPokemon() golurk.Pokemon {
	return golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByPokedex(1)).Build()
}

func getDummyPokemonWithAbility(ability string) golurk.Pokemon {
	pkm := getDummyPokemon()
	pkm.Ability.Name = ability

	return pkm
}

func getSimpleState(playerPkm golurk.Pokemon, enemyPkm golurk.Pokemon) golurk.GameState {
	gameState := golurk.NewState([]golurk.Pokemon{playerPkm}, []golurk.Pokemon{enemyPkm}, golurk.CreateRandomStateSeed())
	return gameState
}

type lowSource struct{}

func (lowSource) Uint64() uint64 {
	return 0
}

type highSource struct{}

func (highSource) Uint64() uint64 {
	return math.MaxUint64
}
