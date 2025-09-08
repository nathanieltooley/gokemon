package tests

import (
	"math"
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

func TestSandstormChip(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = golurk.WEATHER_SANDSTORM

	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	damage := float64(pokemon.MaxHp) * (1.0 / 16.0)
	expectedHp := pokemon.MaxHp - uint(math.Ceil(damage))

	if pokemon.Hp.Value != expectedHp {
		t.Fatalf("pokemon hp did not match expected value. pokemon hp: %d/%d | expected: %d/%d", pokemon.Hp.Value, pokemon.MaxHp, expectedHp, pokemon.MaxHp)
	}
}

func TestSandstormBuff(t *testing.T) {
	sandPokemon := getPerfDummyPokemon("geodude")
	enemyPokemon := getPerfDummyPokemon("bulbasaur")
	// test solar beam debuff and spdef buff
	damage := golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("solar-beam"), false, 0, testingRng)
	checkDamageRange(t, damage, 892, 1056)
	damage = golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("solar-beam"), false, golurk.WEATHER_SANDSTORM, testingRng)
	checkDamageRange(t, damage, 304, 360)

	// test normal special move against spdef buff
	damage = golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("leaf-storm"), false, 0, testingRng)
	checkDamageRange(t, damage, 964, 1140)
	damage = golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("leaf-storm"), false, golurk.WEATHER_SANDSTORM, testingRng)
	checkDamageRange(t, damage, 640, 760)

	// make sure no buff against physical moves
	damage = golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("vine-whip"), false, 0, testingRng)
	checkDamageRange(t, damage, 112, 136)
	damage = golurk.Damage(enemyPokemon, sandPokemon, *golurk.GlobalData.GetMove("vine-whip"), false, golurk.WEATHER_SANDSTORM, testingRng)
	checkDamageRange(t, damage, 112, 136)
}

func TestRainChanges(t *testing.T) {
	waterPokemon := getPerfDummyPokemon("squirtle")
	firePokemon := getPerfDummyPokemon("charmander")

	damage := golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 86, 104)
	damage = golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_RAIN, testingRng)
	checkDamageRange(t, damage, 132, 156)

	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 21, 24)
	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_RAIN, testingRng)
	checkDamageRange(t, damage, 9, 12)
}

func TestSunDamage(t *testing.T) {
	waterPokemon := getPerfDummyPokemon("squirtle")
	firePokemon := getPerfDummyPokemon("charmander")

	damage := golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 86, 104)
	damage = golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_SUN, testingRng)
	checkDamageRange(t, damage, 42, 50)

	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 21, 24)
	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_SUN, testingRng)
	checkDamageRange(t, damage, 30, 36)
}

func TestSunFreeze(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	move := *golurk.GlobalData.GetMove("ice-beam")
	move.Accuracy = 100
	move.Meta.AilmentChance = 100
	move.Power = 0

	enemyPokemon.Moves[0] = move

	gameState := getSimpleState(pokemon, enemyPokemon)
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.AI, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Status != golurk.STATUS_FROZEN {
		t.Fatalf("pokemon was not frozen")
	}

	pokemon.Status = 0

	gameState = getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = golurk.WEATHER_SUN
	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.AI, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Status == golurk.STATUS_FROZEN {
		t.Fatalf("pokemon was frozen in the sun")
	}
}
