package core

import (
	"fmt"
	"math"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog/log"
)

const (
	STAT_ATTACK   = "attack"
	STAT_DEFENSE  = "defense"
	STAT_SPATTACK = "special-attack"
	STAT_SPDEF    = "special-defense"
	STAT_SPEED    = "speed"
	STAT_ACCURACY = "accuracy"
	STAT_EVASION  = "evasion"
)

func StatChangeHandler(state *GameState, pokemon *core.Pokemon, statChange core.StatChange, statChance int) StateSnapshot {
	statCheck := global.GokeRand.IntN(100)
	if statChance == 0 {
		statChance = 100
	}

	statChangeState := StateSnapshot{}

	if statCheck < statChance {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did pass")
		statChangeState.Messages = append(statChangeState.Messages, ChangeStat(pokemon, statChange.StatName, statChange.Change)...)
	} else {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did not pass")
	}

	statChangeState.State = state.Clone()

	return statChangeState
}

func ChangeStat(pokemon *core.Pokemon, statName string, change int) []string {
	messages := make([]string, 0)

	absChange := int(math.Abs(float64(change)))
	if change > 0 {
		messages = append(messages, fmt.Sprintf("%s's %s increased by %d stages!", pokemon.Nickname, statName, absChange))
	} else {
		messages = append(messages, fmt.Sprintf("%s's %s decreased by %d stages!", pokemon.Nickname, statName, absChange))
	}

	// sorry
	switch statName {
	case STAT_ATTACK:
		pokemon.Attack.ChangeStat(change)
	case STAT_DEFENSE:
		pokemon.Def.ChangeStat(change)
	case STAT_SPATTACK:
		pokemon.SpAttack.ChangeStat(change)
	case STAT_SPDEF:
		pokemon.SpDef.ChangeStat(change)
	case STAT_SPEED:
		pokemon.RawSpeed.ChangeStat(change)
	case STAT_ACCURACY:
		pokemon.ChangeAccuracy(change)
	case STAT_EVASION:
		pokemon.ChangeEvasion(change)
	}

	return messages
}

func FlinchHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	pokemon.CanAttackThisTurn = false
	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s flinched!", pokemon.Nickname)},
	}
}

func SleepHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	message := ""

	// Sleep is over
	if pokemon.SleepCount <= 0 {
		pokemon.Status = core.STATUS_NONE
		message = fmt.Sprintf("%s woke up!", pokemon.Nickname)
		pokemon.CanAttackThisTurn = true
	} else {
		message = fmt.Sprintf("%s is asleep", pokemon.Nickname)
		pokemon.CanAttackThisTurn = false
	}

	pokemon.SleepCount--

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{message},
	}
}

func ParaHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	paraChance := 0.5
	paraCheck := global.GokeRand.Float64()

	if paraCheck > paraChance {
		// don't get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check passed")
		return NewEmptyStateSnapshot()
	} else {
		// do get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check failed")
		pokemon.CanAttackThisTurn = false
	}

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s is paralyzed and cannot move", pokemon.Nickname)},
	}
}

func BurnHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	damage := pokemon.MaxHp / 16
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s is burned", pokemon.Nickname), fmt.Sprintf("Burn did %d%% damage", damagePercent)},
	}
}

func PoisonHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	// for future reference, this is MaxHp / 16 in gen 1
	damage := pokemon.MaxHp / 8
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s is poisoned", pokemon.Nickname), fmt.Sprintf("Poison did %d%% damage", damagePercent)},
	}
}

func ToxicHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	damage := (pokemon.MaxHp / 16) * uint(pokemon.ToxicCount)
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	log.Info().Int("toxicCount", pokemon.ToxicCount).Uint("damage", damage).Msg("Toxic Action")

	pokemon.ToxicCount++

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s is badly poisoned", pokemon.Nickname), fmt.Sprintf("Toxic did %d%% damage", damagePercent)},
	}
}

func FreezeHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	thawChance := .20
	thawCheck := global.GokeRand.Float64()

	message := ""

	// pokemon stays frozen
	if thawCheck > thawChance {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check failed")
		message = fmt.Sprintf("%s is frozen and cannot move", pokemon.Nickname)

		pokemon.CanAttackThisTurn = false
	} else {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check succeeded!")
		message = fmt.Sprintf("%s thawed out!", pokemon.Nickname)

		pokemon.Status = core.STATUS_NONE
		pokemon.CanAttackThisTurn = true
	}

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{message},
	}
}

func ConfuseHandler(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	pokemon.ConfusionCount--
	log.Debug().Int("newConfCount", pokemon.ConfusionCount).Msg("confusion lowered")

	confChance := .33
	confCheck := global.GokeRand.Float64()

	// Exit early
	if confCheck > confChance {
		return NewEmptyStateSnapshot()
	}

	confMove := core.Move{
		Name:  "Confusion",
		Power: 40,
		Meta: &core.MoveMeta{
			Category: struct {
				Id   int
				Name string
			}{
				Name: "damage",
			},
		},
		DamageClass: core.DAMAGETYPE_PHYSICAL,
	}

	dmg := Damage(*pokemon, *pokemon, confMove, false, core.WEATHER_NONE)
	pokemon.Damage(dmg)
	pokemon.CanAttackThisTurn = false

	log.Info().Uint("damage", dmg).Msgf("%s hit itself in confusion", pokemon.Nickname)

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s hurt itself in confusion", pokemon.Nickname)},
	}
}

// Single place to apply ailment so that all abilities can be checked
func ApplyAilment(state *GameState, pokemon *core.Pokemon, ailment int) StateSnapshot {
	// If pokemon already has ailment, return early
	if pokemon.Status != core.STATUS_NONE {
		return NewEmptyStateSnapshot()
	}

	pokemon.Status = ailment

	// Post-Ailment initialization
	switch pokemon.Status {
	case core.STATUS_PARA:
		if pokemon.Ability.Name == "limber" {
			pokemon.Status = core.STATUS_NONE
			return NewMessageOnlySnapshot(fmt.Sprintf("%s is Limber and can not be paralyzed!", pokemon.Nickname))
		}
	// Set how many turns the pokemon is asleep for
	case core.STATUS_SLEEP:
		if pokemon.Ability.Name == "insomnia" {
			pokemon.Status = core.STATUS_NONE
			return NewMessageOnlySnapshot(fmt.Sprintf("%s has Insomnia and can not fall asleep!", pokemon.Nickname))
		}

		randTime := global.GokeRand.IntN(2) + 1
		pokemon.SleepCount = randTime
		attackActionLogger().Debug().Msgf("%s is now asleep for %d turns", pokemon.Nickname, pokemon.SleepCount)
	case core.STATUS_POISON:
		if pokemon.Ability.Name == "immunity" {
			pokemon.Status = core.STATUS_NONE
			return NewMessageOnlySnapshot(fmt.Sprintf("%s has Immunity to poison!", pokemon.Nickname))
		}
	case core.STATUS_FROZEN:
		if pokemon.Ability.Name == "magma-armor" {
			pokemon.Status = core.STATUS_NONE
			return NewMessageOnlySnapshot(fmt.Sprintf("%s has Magma Armor and cannot be frozen!", pokemon.Nickname))
		}
	case core.STATUS_TOXIC:
		// Block toxic with immunity
		if pokemon.Ability.Name == "immunity" {
			pokemon.Status = core.STATUS_NONE
			return NewMessageOnlySnapshot(fmt.Sprintf("%s has Immunity to poison!", pokemon.Nickname))
		} else {
			// otherwise init toxic count
			pokemon.ToxicCount = 1
		}
	}

	return NewStateSnapshot(state)
}

func SandstormDamage(state *GameState, pokemon *core.Pokemon) StateSnapshot {
	non_damage_types := []*core.PokemonType{&core.TYPE_ROCK, &core.TYPE_STEEL, &core.TYPE_GROUND}
	non_damage_abilities := []string{"sand-force", "sand-rush", "sand-veil", "magic-guard", "overcoat"}
	if slices.Contains(non_damage_types, pokemon.Base.Type1) || slices.Contains(non_damage_types, pokemon.Base.Type2) {
		return NewEmptyStateSnapshot()
	}

	if slices.Contains(non_damage_abilities, pokemon.Ability.Name) {
		return NewEmptyStateSnapshot()
	}

	if pokemon.Item == "safety-goggles" {
		return NewEmptyStateSnapshot()
	}

	pokemon.DamagePerc(1.0 / 16.0)
	return NewStateSnapshot(state, fmt.Sprintf("%s was damaged by the sandstorm!", pokemon.Nickname))
}
