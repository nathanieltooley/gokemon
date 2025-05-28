package tests

import (
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"

	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
)

func getDummyPokemon() core.Pokemon {
	return game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
}

func getDummyPokemonWithAbility(ability string) core.Pokemon {
	pkm := getDummyPokemon()
	pkm.Ability.Name = ability

	return pkm
}

func getSimpleState(playerPkm core.Pokemon, enemyPkm core.Pokemon) stateCore.GameState {
	gameState := state.NewState([]core.Pokemon{playerPkm}, []core.Pokemon{enemyPkm})
	return gameState
}
