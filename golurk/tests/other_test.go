package tests

import (
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

func TestSpeed(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.RawSpeed.RawValue = 1000
	pokemon.Attack.RawValue = 10000
	enemyPokemon.RawSpeed.RawValue = 1

	pokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")
	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")

	gameState := getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0), golurk.NewAttackAction(golurk.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()

	// A bit goofy of a way to see if speed works but the idea is that
	// the first pokemon is faster and so strong that the second pokemon should move
	// after the first but die before it can attack. If the first pokemon has taken damage, we know
	// that the second pokemon either survived (which is a problem with the damage function) or they attacked first.
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("faster pokemon took damage from slower pokemon when it should not have. hp:%d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	if enemyPokemon.Hp.Value != 0 {
		t.Fatalf("pokemon should have taken fatal damage. hp:%d/%d", enemyPokemon.Hp.Value, enemyPokemon.MaxHp)
	}
}

func TestTaunt(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.RawSpeed.RawValue = 1000

	pokemon.Moves[0] = *golurk.GlobalData.GetMove("taunt")
	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("swords-dance")

	gameState := getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0), golurk.NewAttackAction(golurk.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()

	// pokemon should go before enemyPokemon so taunt should equal 3.
	// end of turn decrements taunt so it should equal 2.
	if enemyPokemon.TauntCount != 2 {
		t.Fatalf("incorrect taunt count. expected 2, got %d", enemyPokemon.TauntCount)
	}

	if enemyPokemon.Attack.Stage != 0 {
		t.Fatalf("taunted pokemon used status move. attack stage: %d", enemyPokemon.Attack.Stage)
	}
}
