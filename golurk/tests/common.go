// Package tests contains integration tests between different packages
package tests

import (
	"fmt"
	"math"
	"os"

	"github.com/nathanieltooley/gokemon/golurk"
)

var (
	testingSeed = golurk.CreateRandomStateSeed()
	testingRng  = golurk.CreateRNG(&testingSeed)
)

func init() {
	dataFiles := os.DirFS("../../poketerm/")
	errs := golurk.DefaultLoader(dataFiles)

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("err: %s\n", err)
		}

		panic("error(s) while trying to load pokemon data")
	}
}

func getDummyPokemon() golurk.Pokemon {
	return golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByPokedex(1), testingRng).Build()
}

func getDummyPokemonWithAbility(ability string) golurk.Pokemon {
	pkm := getDummyPokemon()
	pkm.Ability.Name = ability

	return pkm
}

func getPerfDummyPokemon(pokemonName string) golurk.Pokemon {
	base := golurk.GlobalData.GetPokemonByName(pokemonName)
	return golurk.NewPokeBuilder(base, testingRng).SetPerfectIvs().SetLevel(100).Build()
}

func getSimpleState(playerPkm golurk.Pokemon, enemyPkm golurk.Pokemon) golurk.GameState {
	gameState := golurk.NewState([]golurk.Pokemon{playerPkm}, []golurk.Pokemon{enemyPkm}, golurk.CreateRandomStateSeed())
	return gameState
}

func mustNotBeNil[T any](value *T) T {
	if value == nil {
		panic(value)
	}

	return *value
}

type lowSource struct{}

func (lowSource) Uint64() uint64 {
	return 0
}

type highSource struct{}

func (highSource) Uint64() uint64 {
	return math.MaxUint64
}
