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
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{}))

	pokemonSpeedStage := gameState.HostPlayer.GetActivePokemon().RawSpeed.Stage
	if pokemonSpeedStage != 1 {
		t.Fatalf("pokemon's speed stage is incorrect: got %d expected 1", pokemonSpeedStage)
	}

	enemyPokemonSpeedStage := gameState.ClientPlayer.GetActivePokemon().RawSpeed.Stage
	if enemyPokemonSpeedStage != 0 {
		t.Fatal("enemy pokemon's speed stage boosted incorrectly to 1")
	}

	// Test that pokemon that switch in do not get the speed boost
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.AI, 0)}))

	if gameState.HostPlayer.GetActivePokemon().Hp.Value != 1 {
		t.Fatalf("pokemon should survive with 1 hp because of sturdy, got %d", pokemon.Hp.Value)
	}
}

func TestDamp(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("damp")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("self-destruct")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	if gameState.HostPlayer.GetActivePokemon().Status == core.STATUS_PARA {
		t.Fatal("pokemon with limber was paralyzed")
	}
}

func TestSandVeilSandstormDamage(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("sand-veil")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = core.WEATHER_SANDSTORM

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with volt-absorb took electric type damage")
	}

	pokemon.DamagePerc(.25)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with water-absorb took water type damage")
	}

	pokemon.DamagePerc(.25)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

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

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with flash-fire took fire-type damage: hp %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	damage := stateCore.Damage(pokemon, enemyPokemon, pokemon.Moves[0], false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 108, 128)
}

func TestIntimidate(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("intimidate")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, stateCore.HOST, 0)}))

	intimidatedPokemon := gameState.ClientPlayer.GetActivePokemon()

	if intimidatedPokemon.Attack.Stage != -1 {
		t.Fatalf("pokemon was not intimidated: attack stage = %d", intimidatedPokemon.Attack.Stage)
	}
}

func TestOwnTempo(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("intimidate")
	enemyPokemon := getDummyPokemonWithAbility("own-tempo")

	pokemon.Moves[0] = *global.MOVES.GetMove("teeter-dance")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, stateCore.HOST, 0)}))

	ownTempoPokemon := gameState.ClientPlayer.GetActivePokemon()

	if ownTempoPokemon.Attack.Stage != 0 {
		t.Fatalf("own-tempo pokemon was intimidated: attack stage = %d", ownTempoPokemon.Attack.Stage)
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.HOST, 0)}))

	ownTempoPokemon = gameState.ClientPlayer.GetActivePokemon()

	if ownTempoPokemon.ConfusionCount != 0 {
		t.Fatalf("own-tempo pokemon was confused, count = %d", ownTempoPokemon.ConfusionCount)
	}
}

func TestSuctionCups(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("suction-cups")
	pokemon2 := getDummyPokemon()
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("roar")
	gameState := state.NewState([]core.Pokemon{pokemon, pokemon2}, []core.Pokemon{enemyPokemon})

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	activeIndex := gameState.HostPlayer.ActivePokeIndex
	if activeIndex != 0 {
		t.Fatalf("pokemon with suction cups was switched out! index = %d", activeIndex)
	}
}

func TestWonderGuard(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(5).SetPerfectIvs().Build()
	pokemon.Ability.Name = "wonder-guard"
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("tackle")
	enemyPokemon.Moves[1] = *global.MOVES.GetMove("water-gun")
	enemyPokemon.Moves[2] = *global.MOVES.GetMove("ember")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with wonder guard took damage from non-super effective move. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 1)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with wonder guard took damage from non-super effective move. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 2)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value == pokemon.MaxHp {
		t.Fatalf("pokemon with wonder guard did not take damage from super effective move. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestLevitate(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("levitate")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("earthquake")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with levitate took damage from ground type move. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestHugePower(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()
	pokemon.Ability.Name = "huge-power"
	pokemon.Moves[0] = *global.MOVES.GetMove("tackle")

	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 58, 69)
}

func TestPurePower(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()
	pokemon.Ability.Name = "pure-power"
	pokemon.Moves[0] = *global.MOVES.GetMove("tackle")

	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 58, 69)
}

func TestVitalSpirit(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("vital-spirit")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("spore")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status == core.STATUS_SLEEP {
		t.Fatalf("pokemon with vital spirit fell asleep")
	}
}

func TestWaterVeil(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("water-veil")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("will-o-wisp")
	enemyPokemon.Moves[0].Accuracy = 100

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status == core.STATUS_BURN {
		t.Fatalf("pokemon with water-veil burned")
	}
}

func TestThickFat(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()

	pokemon.Ability.Name = "thick-fat"

	damage := stateCore.Damage(enemyPokemon, pokemon, *global.MOVES.GetMove("flamethrower"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 66, 78)

	damage = stateCore.Damage(enemyPokemon, pokemon, *global.MOVES.GetMove("blizzard"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 80, 96)
}

func TestChloro(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()
	pokemon.Ability.Name = "chlorophyll"

	if pokemon.Speed(core.WEATHER_SUN) != 252 {
		t.Fatalf("pokemon with chlorophyll has the incorrect speed: %d", pokemon.Speed(core.WEATHER_SUN))
	}
}

func TestSwiftSwim(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetPerfectIvs().SetLevel(100).Build()
	pokemon.Ability.Name = "swift-swim"

	if pokemon.Speed(core.WEATHER_RAIN) != 252 {
		t.Fatalf("pokemon with swift-swim has the incorrect speed: %d", pokemon.Speed(core.WEATHER_RAIN))
	}
}

func TestMagmaArmor(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("magma-armor")
	enemyPokemon := getDummyPokemon()

	modIceBeam := *global.MOVES.GetMove("ice-beam")
	modIceBeam.Meta.AilmentChance = 100
	modIceBeam.EffectChance = 100

	enemyPokemon.Moves[0] = modIceBeam

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status == core.STATUS_FROZEN {
		t.Fatalf("pokemon with magma armor froze")
	}
}

func TestLightningRod(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("lightning-rod")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("thunderbolt")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with lightning-rod took damage from an electric-type attack. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}

	if pokemon.SpAttack.Stage != 1 {
		t.Fatalf("pokemon with lightning-rod does not have the right SpAttack stage: got %d, expected 1", pokemon.SpAttack.Stage)
	}
}

func TestPressure(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("pressure")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("tackle")

	startingPP := enemyPokemon.Moves[0].PP

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.PEER, 0)}))

	endingPP := gameState.ClientPlayer.GetActivePokemon().InGameMoveInfo[0].PP
	if endingPP != (startingPP - 2) {
		t.Fatalf("pressure did not take an extra PP (lol) from enemy pokemon: expected %d, got %d", startingPP-2, endingPP)
	}
}

func TestSandStream(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("sand-stream")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)}))

	if gameState.Weather != core.WEATHER_SANDSTORM {
		t.Fatalf("pokemon with sand-stream did not setup sandstorm weather")
	}
}

func TestDrought(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("drought")
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.HOST, 0)}))

	if gameState.Weather != core.WEATHER_SUN {
		t.Fatalf("pokemon with drought did not setup harsh sunlight")
	}
}

func TestRainDish(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("rain-dish")
	pokemon.Hp.Value = 1
	enemyPokemon := getDummyPokemon()

	gameState := getSimpleState(pokemon, enemyPokemon)
	gameState.Weather = core.WEATHER_RAIN

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != 2 {
		t.Fatalf("pokemon with rain-dish did not heal proper amount. got: %d/%d | expected: %d/%d", pokemon.Hp.Value, pokemon.MaxHp, 2, pokemon.MaxHp)
	}
}

func TestKeenEye(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("keen-eye")
	enemyPokemon := getDummyPokemon()

	sattack := *global.MOVES.GetMove("sand-attack")
	sattack.Accuracy = 100
	enemyPokemon.Moves[0] = sattack

	posSAttack := *global.MOVES.GetMove("sand-attack")
	posSAttack.StatChanges = []core.StatChange{{Change: 1, StatName: core.STAT_ACCURACY}}
	pokemon.Moves[0] = posSAttack

	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.PEER, 0), stateCore.NewAttackAction(state.HOST, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.AccuracyStage < 0 {
		t.Fatalf("pokemon with keen eye had accuracy lowered! expected positive change got %d", pokemon.AccuracyStage)
	} else if pokemon.AccuracyStage == 0 {
		t.Fatalf("pokemon with keen eye could not raise it's own accuracy")
	}
}

func TestRockHead(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("rock-head")
	enemyPokemon := getDummyPokemon()

	pokemon.Moves[0] = *global.MOVES.GetMove("double-edge")
	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.HOST, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with rock head took recoil damage. hp: %d/%d", pokemon.Hp.Value, pokemon.MaxHp)
	}
}

func TestInnerFocus(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("inner-focus")
	enemyPokemon := getDummyPokemonWithAbility("intimidate")

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("fake-out")
	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, state.PEER, 0)}))
	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Attack.Stage != 0 {
		t.Fatalf("pokemon with inner focus was intimidated")
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(state.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if !pokemon.CanAttackThisTurn {
		t.Fatalf("pokemon with inner focus flinched")
	}
}

func TestHustleBoost(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()
	pokemon.Ability.Name = "hustle"
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 44, 52)
}

func TestMarvelScale(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()
	pokemon.Ability.Name = "marvel-scale"
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()

	damage := stateCore.Damage(enemyPokemon, pokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)
	pokemon.Status = core.STATUS_PARA
	damageStatus := stateCore.Damage(enemyPokemon, pokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 29, 35)
	checkDamageRange(t, damageStatus, 20, 24)
}

func TestGuts(t *testing.T) {
	pokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()
	pokemon.Ability.Name = "guts"
	enemyPokemon := game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).SetLevel(100).SetPerfectIvs().Build()

	damage := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)
	pokemon.Status = core.STATUS_PARA
	damageStatus := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)
	pokemon.Status = core.STATUS_BURN
	damageBurn := stateCore.Damage(pokemon, enemyPokemon, *global.MOVES.GetMove("tackle"), false, core.WEATHER_NONE)

	checkDamageRange(t, damage, 29, 35)
	checkDamageRange(t, damageStatus, 44, 52)
	checkDamageRange(t, damageBurn, 44, 52)
}

func TestNaturalCure(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("natural-cure")
	enemyPokemon := getDummyPokemon()

	pokemon.Status = core.STATUS_BURN
	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, stateCore.HOST, 0)}))
	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Status != core.STATUS_NONE {
		t.Fatalf("pokemon with natural cure did not cure it's status (naturally)")
	}
}

func TestTrace(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("trace")
	enemyPokemon := getDummyPokemonWithAbility("huge-power")

	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewSwitchAction(&gameState, stateCore.HOST, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Ability.Name != "huge-power" {
		t.Fatalf("pokemon with trace did not gain enemy's ability")
	}
}

func TestTruant(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("truant")
	enemyPokemon := getDummyPokemon()

	pokemon.Moves[0] = *global.MOVES.GetMove("tackle")
	pokemon.Moves[1] = *global.MOVES.GetMove("protect")

	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.TruantShouldActivate {
		t.Fatalf("truant should only activate after the pokemon attacks with a move")
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.HOST, 1)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()

	if !pokemon.TruantShouldActivate {
		t.Fatalf("truant should be primed to activate after a pokemon attacks")
	}

	if enemyPokemon.Hp.Value < enemyPokemon.MaxHp {
		t.Fatalf("pokemon took damage from protect")
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.HOST, 0)}))

	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()
	if enemyPokemon.Hp.Value < enemyPokemon.MaxHp {
		t.Fatalf("opponent of truant took damage on loafing turn!")
	}
}

func TestWhiteSmokeClearBody(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("white-smoke")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("leer")
	pokemon.Moves[0] = *global.MOVES.GetMove("swords-dance")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.HOST, 0), stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Def.Stage != 0 {
		t.Fatalf("pokemon with white smoke had it's stats lowered")
	}

	if pokemon.Attack.Stage <= 0 {
		t.Fatalf("pokemon with white smoke could not raise it's stats")
	}

	pokemon.Ability.Name = "clear-body"

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.HOST, 0), stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Def.Stage != 0 {
		t.Fatalf("pokemon with clear-body had it's stats lowered")
	}

	if pokemon.Attack.Stage <= 0 {
		t.Fatalf("pokemon with clear-body could not raise it's stats")
	}
}

func TestHyperCutter(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("hyper-cutter")
	enemyPokemon := getDummyPokemon()

	modLeer := global.MOVES.GetMove("leer")
	modLeer.StatChanges = []core.StatChange{{Change: -1, StatName: core.STAT_ATTACK}}
	modLeer.EffectChance = 100

	enemyPokemon.Moves[0] = *modLeer
	enemyPokemon.Moves[1] = *global.MOVES.GetMove("leer")

	gameState := getSimpleState(pokemon, enemyPokemon)

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Attack.Stage < 0 {
		t.Fatalf("pokemon with hyper cutter had it's attack lowered")
	}

	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 1)}))
	pokemon = *gameState.HostPlayer.GetActivePokemon()

	if pokemon.Def.Stage != -1 {
		t.Fatalf("pokemon with hyper cutter did not have it's defense lowered!")
	}
}

func TestSynch(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("synchronize")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("nuzzle")

	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()

	if pokemon.Status != enemyPokemon.Status && pokemon.Status == core.STATUS_PARA {
		t.Fatalf("pokemon with synchronize did not sync statuses")
	}
}

func TestLiquidOoze(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("liquid-ooze")
	enemyPokemon := getDummyPokemon()

	drainMove := *global.MOVES.GetMove("tackle")
	drainMove.Meta.Drain = 100
	enemyPokemon.Moves[0] = drainMove

	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	enemyPokemon = *gameState.ClientPlayer.GetActivePokemon()
	if enemyPokemon.Hp.Value == enemyPokemon.MaxHp {
		t.Fatalf("pokemon attacked with drain a pokemon with liquid-ooze and took no self-damage!")
	}
}

func TestSoundproof(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("soundproof")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("hyper-voice")
	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.Hp.Value != pokemon.MaxHp {
		t.Fatalf("pokemon with soundproof took damage from sound-based move")
	}
}

func TestColorChange(t *testing.T) {
	pokemon := getDummyPokemonWithAbility("color-change")
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("tackle")
	gameState := getSimpleState(pokemon, enemyPokemon)
	state.ApplyEventsToState(&gameState, state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)}))

	pokemon = *gameState.HostPlayer.GetActivePokemon()
	if pokemon.BattleType.Name != core.TYPENAME_NORMAL {
		t.Fatalf("pokemon with color change did not change it's type to the attack type of NORMAL")
	}
}
