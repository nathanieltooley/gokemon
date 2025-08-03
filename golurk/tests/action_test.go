package tests

import (
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

func TestForceSwitch(t *testing.T) {
	playerPokemonTeam := []golurk.Pokemon{getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon()}
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("roar")

	gameState := golurk.NewState(playerPokemonTeam, []golurk.Pokemon{enemyPokemon}, golurk.CreateRandomStateSeed())

	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.PEER, 0)}))

	if gameState.HostPlayer.ActivePokeIndex == 0 {
		t.Fatalf("Pokemon not switched out")
	}
}
