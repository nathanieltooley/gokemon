package tests

import (
	"os"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/game/state"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
)

func init() {
	global.GlobalInit(os.DirFS("../"), false)
	global.UpdateLogLevel(zerolog.DebugLevel)
}

// NOTE: Most ability tests will directly set the ability on a pokemon (probably bulbasaur) rather than choosing a pokemon
// with that ability for the simple fact that it really doesn't matter. However it may change if for some reason it's important to the ability
func TestDrizzle(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("drizzle")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)})

	if gameState.Weather != core.WEATHER_RAIN {
		t.Fatalf("Expected weather to be rain: got %d", gameState.Weather)
	}
}

func TestSpeedBoost(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("speed-boost")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)

	if gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage != 0 {
		t.Fatal("test pokemon has started with incorrect speed stage")
	}

	_ = state.ProcessTurn(&gameState, []stateCore.Action{})

	pokemonSpeedStage := gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage is incorrect: got %d expected 1", pokemonSpeedStage)
	}

	enemyPokemonSpeedStage := gameState.ClientPlayer.GetActivePokemon().RawSpeed.Stage
	if enemyPokemonSpeedStage != 0 {
		t.Fatal("enemy pokemon's speed stage boosted incorrectly to 1")
	}

	// Test that pokemon that switch in do not get the speed boost
	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)})

	pokemonSpeedStage = gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage should have stayed at 1: got %d instead", pokemonSpeedStage)
	}
}

func TestSturdy(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("sturdy")
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("charizard")).SetLevel(100).SetPerfectIvs().Build()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("ember")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.AI, 0)})

	if gameState.HostPlayer.GetActivePokemon().Hp.Value != 1 {
		t.Fatalf("pokemon should survive with 1 hp because of sturdy, got %d", pokemon.Hp.Value)
	}
}

func TestDamp(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("damp")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("self-destruct")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp || enemyPokemon.Hp.Value != enemyPokemon.MaxHp {
		t.Fatalf("self-destruct most likely activated. player pokemon hp: %d/%d | enemy pokemon hp: %d|%d", pokemon.Hp.Value, pokemon.MaxHp, enemyPokemon.Hp.Value, enemyPokemon.MaxHp)
	}
}

func TestLimber(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("limber")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("nuzzle")

	gameState := state.NewState([]core.Pokemon{pokemon}, []core.Pokemon{enemyPokemon})

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	if gameState.HostPlayer.GetActivePokemon().Status == core.STATUS_PARA {
		t.Fatal("pokemon with limber was paralyzed")
	}
}

func TestSandVeilSandstormDamage(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("sand-veil")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = core.WEATHER_SANDSTORM

	_ = state.ProcessTurn(&gameState, []stateCore.Action{})

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon most likely took sandstorm chip. pokemon hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestVoltAbsorb(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("volt-absorb")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("spark")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with volt-absorb took electric type damage")
	}

	pokemon.DamagePerc(.25)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with volt-absorb did not heal from electric attack: hp %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestWaterAbsorb(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("water-absorb")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("bubble")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with water-absorb took water type damage")
	}

	pokemon.DamagePerc(.25)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with water-absorb did not heal from water attack: hp %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestInsomnia(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("insomnia")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("spore")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status == core.STATUS_SLEEP {
		t.Fatalf("pokemon with insomnia fell asleep")
	}
}

func TestImmunity(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("immunity")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("toxic")
	enemyPokemon.Moves[0].Accuracy = 100

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status == core.STATUS_POISON || pokemon.Status == core.STATUS_TOXIC {
		t.Fatalf("pokemon with immunity was poisoned")
	}
}

func TestFlashFireImmunity(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("flash-fire")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("flamethrower")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with flash-fire took fire-type damage: hp %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestFlashFire(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByName("vulpix")).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()

	pokemon.Ability.Name = "flash-fire"
	pokemon.Moves[0] = *global.MOVES.GetMove("ember")
	enemyPokemon.Moves[0] = *global.MOVES.GetMove("ember")

	gameState := getSimpleState(pokemon, enemyPokemon)

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with flash-fire took fire-type damage: hp %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	damage := stateCore.Damage(pokemon, enemyPokemon, pokemon.Moves[0], false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 108, 128)
}
