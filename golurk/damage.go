package golurk

import (
	"math"
	"math/rand/v2"

	"github.com/go-logr/logr"
)

var damageLogger = func() logr.Logger {
	return internalLogger.WithName("damage")
}

// Damage calculates the damage an attacking pokemon should do to a defending pokemon
//
// TODO: Rethink this function signature. The amount of arguments is getting a little ridiculous now
func Damage(attacker Pokemon, defendent Pokemon, move Move, crit bool, weather int, rng *rand.Rand) uint {
	attackerLevel := attacker.Level // TODO: Add exception for Beat Up
	var baseA, baseD uint
	var a, d uint // TODO: Add exception for Beat Up
	var aBoost, dBoost int

	// Attack affecting abilities
	switch attacker.Ability.Name {
	case "huge-power", "pure-power":
		attacker.Attack.RawValue *= 2
	case "hustle":
		boostedAtt := math.Round(float64(attacker.Attack.RawValue) * 1.5)
		attacker.Attack.RawValue = uint(boostedAtt)
	}

	if defendent.Ability.Name == "marvel-scale" && defendent.Status != STATUS_NONE {
		boostedDef := math.Round(float64(defendent.Def.RawValue) * 1.5)
		defendent.Def.RawValue = uint(boostedDef)
	}

	// Determine damage type
	switch move.DamageClass {
	case DAMAGETYPE_PHYSICAL:
		baseA = attacker.Attack.RawValue
		a = uint(attacker.Attack.CalcValue())
		aBoost = attacker.Attack.Stage

		baseD = defendent.Def.RawValue
		d = uint(defendent.Def.CalcValue())
		dBoost = defendent.Def.Stage
	case DAMAGETYPE_SPECIAL:
		baseA = attacker.SpAttack.RawValue
		a = uint(attacker.SpAttack.CalcValue())
		aBoost = attacker.SpAttack.Stage

		baseD = defendent.SpDef.RawValue
		d = uint(defendent.SpDef.CalcValue())
		dBoost = defendent.SpDef.Stage
	}

	flashFireBoost := 1.0
	if attacker.FlashFire {
		flashFireBoost = 1.5
	}

	// Boost attack or special attack while flash-fire boosted and using fire attack
	if move.Type == TYPENAME_FIRE {
		a = uint(float64(a) * flashFireBoost)
	}

	power := move.Power

	if power == 0 {
		return 0
	}

	damageLogger().V(2).Info("Type 1", "type", defendent.Base.Type1)
	damageLogger().V(2).Info("Type 2", "type", defendent.Base.Type2)

	attackType := GetAttackTypeMapping(move.Type)

	effectiveness := defendent.DefenseEffectiveness(attackType)

	if effectiveness == 0 {
		return 0
	}

	if defendent.Ability.Name == "wonder_guard" {
		if effectiveness <= 1 {
			return 0
		}
	}

	if defendent.Ability.Name == "levitate" && move.Type == TYPENAME_GROUND {
		return 0
	}

	if defendent.Ability.Name == "lightning-rod" && move.Type == TYPENAME_ELECTRIC {
		return 0
	}

	var critBoost float64 = 1
	if crit && defendent.Ability.Name != "battle-armor" && defendent.Ability.Name != "shell-armor" {
		critBoost = 1.5
		a = baseA
		d = baseD

	} else if crit && (defendent.Ability.Name == "battle-armor" || defendent.Ability.Name == "shell-armor") {
		damageLogger().Info("Defending pokemon blocked crit", "pokemon_name", defendent.Nickname, "ability_name", defendent.Ability.Name)
	}

	lowHealthBonus := 1.0
	if float32(attacker.Hp.Value) <= float32(attacker.MaxHp)*0.33 {
		if (attacker.Ability.Name == "overgrow" && move.Type == TYPENAME_GRASS) ||
			(attacker.Ability.Name == "blaze" && move.Type == TYPENAME_FIRE) ||
			(attacker.Ability.Name == "torrent" && move.Type == TYPENAME_WATER) ||
			(attacker.Ability.Name == "swarm" && move.Type == TYPENAME_BUG) {
			lowHealthBonus = 1.5
		}
	}

	a = uint(float64(a) * lowHealthBonus)

	if defendent.Ability.Name == "thick-fat" {
		if move.Type == TYPENAME_ICE || move.Type == TYPENAME_FIRE {
			a = uint(float64(a) * 0.5)
		}
	}

	var burn float64 = 1
	// TODO: Add exception for Guts and Facade
	if attacker.Status == STATUS_BURN && move.DamageClass == DAMAGETYPE_PHYSICAL {
		burn = 0.5
		damageLogger().V(2).Info("Attacker is burned and is using a physical move", "attacker_name", attacker.Nickname)
	}

	if attacker.Ability.Name == "guts" {
		// remove burn debuff
		burn = 1
		if attacker.Status != STATUS_NONE {
			a = uint(float64(a) * 1.5)
		}
	}

	// Calculate the part of the damage function in brackets
	// TODO: still has rounding issues. not sure if its here in the order of floors and rounds
	// or if later on where a certain value is supposed to be floored or rounded. its really dumb and confusing
	damageInner := math.Floor(math.Floor(math.Floor((float64(2*attackerLevel)/5+2)*float64(power))*(float64(a)/float64(d)))/50 + 2)
	randomSpread := float64(rng.UintN(16)+85) / 100.0
	var stab float64 = 1

	if move.Type == attacker.Base.Type1.Name || (attacker.Base.Type2 != nil && move.Type == attacker.Base.Type2.Name) {
		stab = 1.5
	}

	weatherBonus := 1.0
	if (weather == WEATHER_RAIN && move.Type == TYPENAME_WATER) || (weather == WEATHER_SUN && move.Type == TYPENAME_FIRE) {
		weatherBonus = 1.5
	}
	if (weather == WEATHER_RAIN && move.Type == TYPENAME_FIRE) || (weather == WEATHER_SUN && move.Type == TYPENAME_WATER) {
		weatherBonus = 0.5
	}

	// TODO: Maybe check for parental bond, glaive rush, targets in DBs, ZMoves

	// TODO: There are about 20 different moves, abilities, and items that have some sort of
	// random effect to the damage calcs. Maybe implement the most impactful ones?

	damage := damageInner
	damage = pokeRound(damage * weatherBonus)
	damage = math.Floor(damage * critBoost)
	damage = math.Floor(damage * randomSpread)
	damage = pokeRound(damage * stab)
	damage = math.Floor(damage * effectiveness)
	damage = pokeRound(damage * burn)
	damage = pokeRound(damage * lowHealthBonus)

	finalDamage := uint(damage)

	damageLogger().Info("final damage",
		"power", power,
		"attackerLevel", attackerLevel,
		"attackValue", a,
		"attackChange", aBoost,
		"defValue", d,
		"defenseChange", dBoost,
		"attackType", move.Type,
		"lowHealthBonus", lowHealthBonus,
		"damageInner", damageInner,
		"randomSpread", randomSpread,
		"STAB", stab,
		"Net Type Effectiveness", effectiveness,
		"crit", critBoost,
		"weatherBonus", weatherBonus,
		"flashFire", flashFireBoost,
		"damage", finalDamage)

	return finalDamage
}

func pokeRound(x float64) float64 {
	intPart := math.Trunc(x)
	distance := math.Abs(x - intPart)

	if distance > 0.5 {
		// Would use something like Copysign but this will only deal with positive numbers
		return intPart + 1
	} else {
		return intPart
	}
}
