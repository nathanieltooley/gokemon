package state

import (
	"os"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
)

func init() {
	global.GlobalInit(os.DirFS("../../../"), false)
}

func mustNotBeNil[T any](value *T) T {
	if value == nil {
		panic(value)
	}

	return *value
}

func TestAttackMove(t *testing.T) {
	playerPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("Bulbasaur")).Build()
	aiPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("Charizard")).Build()

	aiPokemon.Moves[0] = mustNotBeNil(global.MOVES.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(global.MOVES.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(global.MOVES.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(global.MOVES.GetMove("scary-face"))

	gameState := NewState([]core.Pokemon{playerPokemon}, []core.Pokemon{aiPokemon})

	aiResult := BestAiAction(&gameState)

	if aiResult != stateCore.NewAttackAction(AI, 1) {
		t.Fatalf("Attack move should be ember, got: %+v", aiPokemon.Moves[aiResult.(stateCore.AttackAction).AttackerMove])
	}
}

func TestSlowMove(t *testing.T) {
	playerPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("Bulbasaur")).SetIvs([6]uint{0, 0, 0, 0, 0, 252}).Build()
	aiPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("Charmander")).Build()

	aiPokemon.Moves[0] = mustNotBeNil(global.MOVES.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(global.MOVES.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(global.MOVES.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(global.MOVES.GetMove("scary-face"))

	gameState := NewState([]core.Pokemon{playerPokemon}, []core.Pokemon{aiPokemon})

	aiResult := BestAiAction(&gameState)

	if aiResult != stateCore.NewAttackAction(AI, 3) {
		t.Logf("pSpeed: %d | aSpeed: %d", playerPokemon.Speed(), aiPokemon.Speed())
		t.Fatalf("Attack move should be scary-face, got: %+v", aiPokemon.Moves[aiResult.(stateCore.AttackAction).AttackerMove])
	}
}
