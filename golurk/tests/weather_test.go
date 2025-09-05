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
	sandPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("geodude"), testingRng).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()

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
	waterPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("squirtle"), testingRng).SetPerfectIvs().SetLevel(100).Build()
	firePokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("charmander"), testingRng).SetPerfectIvs().SetLevel(100).Build()

	damage := golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 86, 104)
	damage = golurk.Damage(waterPokemon, firePokemon, *golurk.GlobalData.GetMove("water-gun"), false, golurk.WEATHER_RAIN, testingRng)
	checkDamageRange(t, damage, 132, 156)

	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_NONE, testingRng)
	checkDamageRange(t, damage, 21, 24)
	damage = golurk.Damage(firePokemon, waterPokemon, *golurk.GlobalData.GetMove("ember"), false, golurk.WEATHER_RAIN, testingRng)
	checkDamageRange(t, damage, 9, 12)
}
