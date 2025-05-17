package core

import (
	"math"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var damageLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-damage").Logger()
	return &logger
}

func Damage(attacker core.Pokemon, defendent core.Pokemon, move core.Move, crit bool, weather int) uint {
	attackerLevel := attacker.Level // TODO: Add exception for Beat Up
	var baseA, baseD uint
	var a, d uint // TODO: Add exception for Beat Up
	var aBoost, dBoost int

	if attacker.Ability.Name == "huge-power" {
		attacker.Attack.RawValue *= 2
	}

	// Determine damage type
	switch move.DamageClass {
	case core.DAMAGETYPE_PHYSICAL:
		baseA = attacker.Attack.RawValue
		a = uint(attacker.Attack.CalcValue())
		aBoost = attacker.Attack.Stage

		baseD = defendent.Def.RawValue
		d = uint(defendent.Def.CalcValue())
		dBoost = defendent.Def.Stage
	case core.DAMAGETYPE_SPECIAL:
		baseA = attacker.SpAttack.RawValue
		a = uint(attacker.SpAttack.CalcValue())
		aBoost = attacker.SpAttack.Stage

		baseD = defendent.SpDef.RawValue
		d = uint(defendent.SpDef.CalcValue())
		dBoost = defendent.SpDef.Stage
	}

	power := move.Power

	if power == 0 {
		return 0
	}

	damageLogger().Debug().Msgf("Type 1: %#v", defendent.Base.Type1)
	damageLogger().Debug().Msgf("Type 2: %#v", defendent.Base.Type2)

	attackType := core.GetAttackTypeMapping(move.Type)

	var type1Effectiveness float32 = 1
	var type2Effectiveness float32 = 1

	if attackType != nil {
		type1Effectiveness = attackType.AttackEffectiveness(defendent.Base.Type1.Name)

		if defendent.Base.Type2 != nil {
			type2Effectiveness = attackType.AttackEffectiveness(defendent.Base.Type2.Name)
		}
	}

	if type1Effectiveness == 0 || type2Effectiveness == 0 {
		return 0
	}

	if defendent.Ability.Name == "wonder_guard" {
		if type1Effectiveness <= 1 && type2Effectiveness <= 1 {
			return 0
		}
	}

	var critBoost float32 = 1
	if crit && defendent.Ability.Name != "battle-armor" && defendent.Ability.Name != "shell-armor" {
		critBoost = 1.5
		a = baseA
		d = baseD

	} else if crit && (defendent.Ability.Name == "battle-armor" || defendent.Ability.Name == "shell-armor") {
		damageLogger().Info().Msgf("Defending pokemon blocked crits with %s", defendent.Ability.Name)
	}

	var lowHealthBonus float32 = 1.0
	if float32(attacker.Hp.Value) <= float32(attacker.MaxHp)*0.33 {
		if (attacker.Ability.Name == "overgrow" && move.Type == core.TYPENAME_GRASS) ||
			(attacker.Ability.Name == "blaze" && move.Type == core.TYPENAME_FIRE) ||
			(attacker.Ability.Name == "torrent" && move.Type == core.TYPENAME_WATER) ||
			(attacker.Ability.Name == "swarm" && move.Type == core.TYPENAME_BUG) {
			lowHealthBonus = 1.5
		}
	}

	a = uint(float32(a) * lowHealthBonus)

	// Calculate the part of the damage function in brackets
	damageInner := ((((float32(2*attackerLevel) / 5) + 2) * float32(power) * (float32(a) / float32(d))) / 50) + 2
	randomSpread := float32(global.GokeRand.UintN(15)+85) / 100
	var stab float32 = 1

	if move.Type == attacker.Base.Type1.Name || (attacker.Base.Type2 != nil && move.Type == attacker.Base.Type2.Name) {
		stab = 1.5
	}

	var burn float32 = 1
	// TODO: Add exception for Guts and Facade
	if attacker.Status == core.STATUS_BURN && move.DamageClass == core.DAMAGETYPE_PHYSICAL {
		burn = 0.5
		damageLogger().Info().Float32("burn", burn).Msg("Attacker is burned and is using a physical move")
	}

	// TODO: Maybe add weather
	var weatherBonus float32 = 1.0
	if (weather == core.WEATHER_RAIN && move.Type == core.TYPENAME_WATER) || (weather == core.WEATHER_SUN && move.Type == core.TYPENAME_FIRE) {
		weatherBonus = 1.5
	}
	if (weather == core.WEATHER_RAIN && move.Type == core.TYPENAME_FIRE) || (weather == core.WEATHER_SUN && move.Type == core.TYPENAME_WATER) {
		weatherBonus = 0.5
	}

	// TODO: Maybe check for parental bond, glaive rush, targets in DBs, ZMoves

	// TODO: There are about 20 different moves, abilities, and items that have some sort of
	// random effect to the damage calcs. Maybe implement the most impactful ones?

	// This seems to mostly work, however there are issues when it comes to rounding
	// and it seems that the lowest possible value in a damage range may not be able
	// to show up as often because rounding is a bit different
	// TODO: maybe make a custom rounding function that rounds DOWN at .5
	damage := uint(math.Round(float64(damageInner * randomSpread * type1Effectiveness * type2Effectiveness * stab * burn * critBoost * weatherBonus)))

	damageLogger().Debug().
		Int("power", power).
		Uint("attackerLevel", attackerLevel).
		Uint("attackValue", a).
		Int("attackChange", aBoost).
		Uint("defValue", d).
		Int("defenseChange", dBoost).
		Str("attackType", move.Type).
		Float32("lowHealthBonus", lowHealthBonus).
		Float32("damageInner", damageInner).
		Float32("randomSpread", randomSpread).
		Float32("STAB", stab).
		Float32("Net Type Effectiveness", type1Effectiveness*type2Effectiveness).
		Float32("crit", critBoost).
		Float32("weatherBonus", weatherBonus).
		Uint("damage", damage).
		Msg("damage calc")

	return damage
}
