package golurk

import (
	"testing"
)

func mustNotBeNil[T any](value *T) T {
	if value == nil {
		panic(value)
	}

	return *value
}

func TestAttackMove(t *testing.T) {
	playerPokemon := NewPokeBuilder(internalData.GetPokemonByName("Bulbasaur"), internalRng).Build()
	aiPokemon := NewPokeBuilder(internalData.GetPokemonByName("Charizard"), internalRng).Build()

	aiPokemon.Moves[0] = mustNotBeNil(internalData.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(internalData.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(internalData.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(internalData.GetMove("scary-face"))

	// do go tests run in parallel and are RNG impls thread-safe. asking for a friend
	gameState := NewState([]Pokemon{playerPokemon}, []Pokemon{aiPokemon}, internalSeed)

	aiResult := BestAiAction(&gameState)

	if aiResult != NewAttackAction(AI, 1) {
		t.Fatalf("Attack move should be ember, got: %+v", aiPokemon.Moves[aiResult.(AttackAction).AttackerMove])
	}
}

func TestSlowMove(t *testing.T) {
	playerPokemon := NewPokeBuilder(internalData.GetPokemonByName("Bulbasaur"), internalRng).SetIvs([6]uint{0, 0, 0, 0, 0, 252}).Build()
	aiPokemon := NewPokeBuilder(internalData.GetPokemonByName("Charmander"), internalRng).Build()

	aiPokemon.Moves[0] = mustNotBeNil(internalData.GetMove("tackle"))
	aiPokemon.Moves[1] = mustNotBeNil(internalData.GetMove("ember"))
	aiPokemon.Moves[2] = mustNotBeNil(internalData.GetMove("tail-whip"))
	aiPokemon.Moves[3] = mustNotBeNil(internalData.GetMove("scary-face"))

	gameState := NewState([]Pokemon{playerPokemon}, []Pokemon{aiPokemon}, CreateRandomStateSeed())

	aiResult := BestAiAction(&gameState)

	if aiResult != NewAttackAction(AI, 3) {
		t.Logf("pSpeed: %d | aSpeed: %d", playerPokemon.Speed(0), aiPokemon.Speed(0))
		t.Fatalf("Attack move should be scary-face, got: %+v", aiPokemon.Moves[aiResult.(AttackAction).AttackerMove])
	}
}
