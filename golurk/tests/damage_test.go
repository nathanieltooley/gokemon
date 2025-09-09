package tests

import (
	"math/rand/v2"
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
)

const iterCount = 1000

func TestDamage(t *testing.T) {
	for range iterCount {
		pokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()
		enemyPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()

		pokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")

		damage := golurk.Damage(pokemon, enemyPokemon, pokemon.Moves[0], false, golurk.WEATHER_NONE, testingRng)

		checkDamageRange(t, damage, 29, 35)
	}
}

func TestDamageLow(t *testing.T) {
	pokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()

	damage := golurk.Damage(pokemon, enemyPokemon, *golurk.GlobalData.GetMove("tackle"), false, golurk.WEATHER_NONE, rand.New(lowSource{}))

	if damage != 29 {
		t.Fatalf("low damage incorrect: expected 29, got %d", damage)
	}
}

func TestDamageHigh(t *testing.T) {
	pokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()

	damage := golurk.Damage(pokemon, enemyPokemon, *golurk.GlobalData.GetMove("tackle"), false, golurk.WEATHER_NONE, rand.New(highSource{}))

	if damage != 35 {
		t.Fatalf("high damage incorrect: expected 35, got %d", damage)
	}
}

func TestCritDamage(t *testing.T) {
	for range iterCount {
		pokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()
		enemyPokemon := golurk.NewPokeBuilder(golurk.GlobalData.GetPokemonByName("bulbasaur"), testingRng).SetPerfectIvs().SetLevel(100).Build()

		pokemon.Moves[0] = *golurk.GlobalData.GetMove("tackle")

		damage := golurk.Damage(pokemon, enemyPokemon, pokemon.Moves[0], true, golurk.WEATHER_NONE, testingRng)

		checkDamageRange(t, damage, 44, 52)
	}
}

func TestBattleTypes(t *testing.T) {
	pokemon := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	pokemon.BattleType = &golurk.TYPE_FLYING

	enemyPokemon.Moves[0] = *golurk.GlobalData.GetMove("earthquake")

	gameState := getSimpleState(pokemon, enemyPokemon)

	golurk.ApplyEventsToState(&gameState, golurk.ProcessTurn(&gameState, []golurk.Action{golurk.NewAttackAction(golurk.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with battle type flying took damage from ground attack")
	}
}

func checkDamageRange(t *testing.T, damage uint, low uint, high uint) {
	if damage < low || damage > high {
		t.Fatalf("outside damage range: should be between %d - %d, got %d", low, high, damage)
	}
}
