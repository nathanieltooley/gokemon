package tests

import (
	"reflect"
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

func TestSpeed(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")
	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")

	gameState := getSimpleState(pokemon, enemyPokemon)
	editPokemon := gameState.HostPlayer.GetActivePokemon()
	editPokemon.RawSpeed.RawValue = 1000
	editPokemon.Attack.RawValue = 10000

	gameState.ClientPlayer.GetActivePokemon().RawSpeed.RawValue = 1
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

	pokemon.Moves[0] = *golurk.GlobalData.GetMove("taunt")
	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("swords-dance")

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.HostPlayer.GetActivePokemon().RawSpeed.RawValue = 1000
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

func TestInfatuation(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.Gender = "male"
	pokemon.Moves[0] = *golurk.GlobalData.GetMove("attract")
	enemyPokemon.Gender = "female"

	gameState := getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()

	if enemyPokemon.InfatuationTarget != 0 {
		t.Fatalf("pokemon failed to become infatuated")
	}

	pokemon.Gender = "female"
	enemyPokemon.InfatuationTarget = -1

	gameState = getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0)}))

	if enemyPokemon.InfatuationTarget != -1 {
		t.Fatalf("same-sex infatuation is illegal in the pokemon world")
	}

	pokemon.Gender = "unknown"
	enemyPokemon.InfatuationTarget = -1

	gameState = getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0)}))

	if enemyPokemon.InfatuationTarget != -1 {
		t.Fatalf("nonbinary host pokemon infatuated binary client")
	}

	pokemon.Gender = "male"
	enemyPokemon.Gender = "unknown"
	enemyPokemon.InfatuationTarget = -1

	gameState = getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.HOST, 0)}))

	if enemyPokemon.InfatuationTarget != -1 {
		t.Fatalf("binary host pokemon infatuated nonbinary client")
	}
}

func TestValidation(t *testing.T) {
	pokemon := getDummyPokemon()
	pokemon.Hp.Ev = 999
	pokemon.Attack.Ev = 999
	pokemon.Def.Ev = 999
	pokemon.SpDef.Ev = 999
	pokemon.SpAttack.Ev = 999
	pokemon.RawSpeed.Ev = 999

	pokemon.Hp.Iv = 999
	pokemon.Attack.Iv = 999
	pokemon.Def.Iv = 999
	pokemon.SpDef.Iv = 999
	pokemon.SpAttack.Iv = 999
	pokemon.RawSpeed.Iv = 999

	pokemon.Level = 999
	pokemon.MaxHp = 999

	pokemon.Init()

	basePokemon := getDummyPokemon()
	basePokemon.Hp.Ev = 252
	basePokemon.Attack.Ev = 252
	basePokemon.Def.Ev = 6
	basePokemon.SpAttack.Ev = 0
	basePokemon.SpDef.Ev = 0
	basePokemon.RawSpeed.Ev = 0

	basePokemon.Hp.Iv = 31
	basePokemon.Attack.Iv = 31
	basePokemon.Def.Iv = 31
	basePokemon.SpAttack.Iv = 31
	basePokemon.SpDef.Iv = 31
	basePokemon.RawSpeed.Iv = 31

	basePokemon.Level = 100

	basePokemon.Init()
	basePokemon.ReCalcStats()

	if !reflect.DeepEqual(pokemon, basePokemon) {
		t.Fatalf("incorrect validation. expected %+v \n\n got %+v", basePokemon, pokemon)
	}
}
