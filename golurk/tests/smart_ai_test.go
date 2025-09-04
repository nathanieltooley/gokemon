package tests

import (
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

func TestAttackMove(t *testing.T) {
	playerPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).Build()
	aiPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("charizard"), testingRng).Build()

	aiPokemon.Moves[0] = mustNotBeNil(golurk.GlobalData.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(golurk.GlobalData.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(golurk.GlobalData.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(golurk.GlobalData.GetMove("scary-face"))

	// do Go tests run in parallel and are RNG impls thread-safe? asking for a friend
	gameState := golurk.NewState([]golurk.Pokemon{playerPokemon}, []golurk.Pokemon{aiPokemon}, testingSeed)

	aiResult := golurk.BestAiAction(&gameState)

	if aiResult != golurk.NewAttackAction(golurk.AI, 1) {
		t.Fatalf("Attack move should be ember, got: %+v", aiPokemon.Moves[aiResult.(golurk.AttackAction).AttackerMove])
	}
}

func TestSlowMove(t *testing.T) {
	playerPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetIvs([6]uint{0, 0, 0, 0, 0, 252}).Build()
	aiPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("charmander"), testingRng).Build()

	aiPokemon.Moves[0] = mustNotBeNil(golurk.GlobalData.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(golurk.GlobalData.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(golurk.GlobalData.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(golurk.GlobalData.GetMove("scary-face"))

	gameState := golurk.NewState([]golurk.Pokemon{playerPokemon}, []golurk.Pokemon{aiPokemon}, golurk.CreateRandomStateSeed())

	aiResult := golurk.BestAiAction(&gameState)

	if aiResult != golurk.NewAttackAction(golurk.AI, 3) {
		t.Logf("pSpeed: %d | aSpeed: %d", playerPokemon.Speed(0), aiPokemon.Speed(0))
		t.Fatalf("Attack move should be scary-face, got: %+v", aiPokemon.Moves[aiResult.(golurk.AttackAction).AttackerMove])
	}
}
