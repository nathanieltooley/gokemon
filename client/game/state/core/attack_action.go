package core

import (
	"fmt"
	"math"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var attackActionLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "attack-action").Logger()
	return &logger
}

var struggleMove = core.Move{
	Accuracy:    100,
	DamageClass: "physical",
	Meta: &core.MoveMeta{
		Category: struct {
			Id   int
			Name string
		}{
			Name: "damage",
		},
		Drain: -25,
	},
	Power: 50,
	Type:  "typeless",
	Name:  "struggle",
}

type AttackAction struct {
	Ctx ActionCtx

	AttackerMove int
}

func NewAttackAction(attacker int, attackMove int) AttackAction {
	return AttackAction{
		Ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

var targetsAffectedByEvasion = [...]string{"specific-move", "selected-pokemon-me-first", "random-opponent", "all-other-pokemon", "selected-pokemon", "all-opponents", "entire-field", "all-pokemon", "fainting-pokemon"}

func (a AttackAction) UpdateState(state GameState) []StateSnapshot {
	attacker := state.GetPlayer(a.Ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.Ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	var move core.Move
	var moveVars core.BattleMove
	var pp uint

	if a.AttackerMove == -1 {
		move = struggleMove
		pp = 1
	} else {
		// TODO: Make sure a.AttackerMove is between 0 -> 3
		move = attackPokemon.Moves[a.AttackerMove]
		moveVars = attackPokemon.InGameMoveInfo[a.AttackerMove]
		pp = moveVars.PP
	}

	states := make([]StateSnapshot, 0)

	// "hack" to show this message first
	useMoveState := StateSnapshot{}
	useMoveState.State = state.Clone()
	useMoveState.Messages = append(useMoveState.Messages, fmt.Sprintf("%s used %s", attackPokemon.Nickname, move.Name))

	states = append(states, useMoveState)

	accuracyCheck := global.GokeRand.IntN(100)

	moveAccuracy := move.Accuracy
	if moveAccuracy == 0 {
		moveAccuracy = 100
	}

	var effectiveEvasion float32 = 1.0
	if lo.Contains(targetsAffectedByEvasion[0:], move.Target.Name) {
		effectiveEvasion = defPokemon.Evasion()
	}

	accuracy := int(float32(moveAccuracy) * (attackPokemon.Accuracy() * effectiveEvasion))

	if state.Weather == core.WEATHER_SANDSTORM && defPokemon.Ability.Name == "sand-veil" {
		accuracy = int(float32(accuracy) * 0.8)
	} else if attackPokemon.Ability.Name == "compound-eyes" && move.Meta.Category.Name != "ohko" {
		accuracy = int(float32(accuracy) * 1.3)
	}

	if accuracyCheck < accuracy && pp > 0 {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", accuracy).Msg("Check passed")

		defImmune := false

		// TODO: This doesn't activate through protect!
		if move.Type == core.TYPENAME_ELECTRIC && defPokemon.Ability.Name == "volt-absorb" {
			states = append(states, NewMessageOnlySnapshot(
				fmt.Sprintf("%s activated volt absorb!", defPokemon.Nickname),
				fmt.Sprintf("%s healed 25%%!", defPokemon.Nickname)),
			)

			defPokemon.HealPerc(.25)
			defImmune = true
		}

		// TODO: This doesn't activate through protect!
		if move.Type == core.TYPENAME_WATER && defPokemon.Ability.Name == "water-absorb" {
			states = append(states, NewMessageOnlySnapshot(
				fmt.Sprintf("%s activated Water Absorb!", defPokemon.Nickname),
				fmt.Sprintf("%s healed 25%%!", defPokemon.Nickname)),
			)

			defPokemon.HealPerc(.25)
			defImmune = true
		}

		// TODO: This doesn't activate through protect or while frozen!
		// TODO: The boost doesn't pass with baton-pass!
		if move.Type == core.TYPENAME_FIRE && defPokemon.Ability.Name == "flash-fire" {
			states = append(states, NewMessageOnlySnapshot(
				fmt.Sprintf("%s activated Flash-Fire", defPokemon.Nickname),
				fmt.Sprintf("%s boosted it's fire-type attacks!", defPokemon.Nickname),
			))

			defPokemon.FlashFire = true
			defImmune = true
		}

		if defPokemon.Ability.Name == "damp" {
			if slices.Contains(core.EXPLOSIVE_MOVES, move.Name) {
				states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s prevented %s with their ability: Damp", defPokemon.Nickname, move.Name)))
				defImmune = true
			}
		}

		if !defImmune {
			// TODO: handle these categories
			// - swagger
			// - unique
			switch move.Meta.Category.Name {
			case "damage", "damage+heal":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
			case "ailment":
				states = append(states, ailmentHandler(&state, defPokemon, move))
			case "damage+ailment":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
				states = append(states, ailmentHandler(&state, defPokemon, move))
			case "net-good-stats":
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					// since its "net-good-stats", the stat change always has to benefit the user
					affectedPokemon := attackPokemon
					if statChange.Change < 0 {
						affectedPokemon = defPokemon
					}

					states = append(states, StatChangeHandler(&state, affectedPokemon, statChange, move.Meta.StatChance))
				})
			// Damages and then CHANGES the targets stats
			case "damage+lower":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					states = append(states, StatChangeHandler(&state, defPokemon, statChange, move.Meta.StatChance))
				})
			// Damages and then CHANGES the user's stats
			// this is different from what pokeapi says (raises instead of changes)
			// and this is important because moves like draco-meteor and overheat
			// lower the user's stats but are in this category
			case "damage+raise":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					states = append(states, StatChangeHandler(&state, attackPokemon, statChange, move.Meta.StatChance))
				})
			case "heal":
				states = append(states, healHandler(&state, attackPokemon, move))
			case "ohko":
				states = append(states, ohkoHandler(&state, attackPokemon, defPokemon))
			case "force-switch":
				states = append(states, forceSwitchHandler(&state, defender))
			default:
				attackActionLogger().Warn().Msgf("Move, %s (%s category), has no handler!!!", move.Name, move.Meta.Category.Name)
			}
		}

		flinchChance := move.Meta.FlinchChance
		if flinchChance == 0 && attackPokemon.Ability.Name == "stench" {
			flinchChance = 10
		}

		if defPokemon.Ability.Name == "inner-focus" {
			flinchChance = 0
			states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s can not be flinched", defPokemon.Nickname)))
		}

		if global.GokeRand.IntN(100) < flinchChance {
			states = append(states, FlinchHandler(&state, defPokemon))
		}

		ppModifier := 1

		// -1 is struggle
		if a.AttackerMove != -1 {
			// TODO: this check will have to change for double battles (any opposing pokemon not the defending)
			if defPokemon.Ability.Name == "pressure" {
				ppModifier = 2
			}

			attackPokemon.InGameMoveInfo[a.AttackerMove].PP = pp - uint(ppModifier)
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", accuracy).Msg("Check failed")
		states = append(states, StateSnapshot{
			State:    state.Clone(),
			Messages: []string{"It Missed!"},
		})
	}

	finalState := StateSnapshot{
		State: state.Clone(),
	}
	states = append(states, finalState)

	return states
}

func damageMoveHandler(state *GameState, attackPokemon *core.Pokemon, defPokemon *core.Pokemon, move core.Move) []StateSnapshot {
	states := make([]StateSnapshot, 0)
	crit := false

	if global.GokeRand.Float32() < attackPokemon.CritChance() {
		crit = true
		log.Info().Float32("chance", attackPokemon.CritChance()).Msg("Attack crit!")
	}

	effectiveness := defPokemon.Base.DefenseEffectiveness(core.GetAttackTypeMapping(move.Type))

	if crit && (defPokemon.Ability.Name == "battle-armor" || defPokemon.Ability.Name == "shell-armor") {
		states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s cannot be crit!", defPokemon.Nickname)))
		crit = false
	}

	damage := Damage(*attackPokemon, *defPokemon, move, crit, state.Weather)

	if defPokemon.Ability.Name == "sturdy" {
		if damage >= defPokemon.Hp.Value && defPokemon.Hp.Value == defPokemon.MaxHp {
			// set the defending pokemon's hp to 1
			damage = defPokemon.MaxHp - 1
			states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s activated sturdy and held on!", defPokemon.Nickname)))
		}
	}

	if defPokemon.Ability.Name == "wonder-guard" {
		if effectiveness < 2 {
			states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s is protected by Wonder Guard!", defPokemon.Nickname)))
			return states
		}
	}

	if defPokemon.Ability.Name == "lightning-rod" && move.Type == core.TYPENAME_ELECTRIC {
		states = append(states, StatChangeHandler(state, defPokemon, core.StatChange{Change: 1, StatName: STAT_SPATTACK}, 100))
	}

	defPokemon.Damage(damage)

	log.Info().Msgf("%s attacked %s, dealing %d damage", attackPokemon.Nickname, defPokemon.Nickname, damage)

	attackActionLogger().Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	attackPercent := uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	damageState := StateSnapshot{}

	if crit {
		damageState.Messages = append(damageState.Messages, fmt.Sprintf("%s critically hit!", attackPokemon.Nickname))
	}
	damageState.Messages = append(damageState.Messages, fmt.Sprintf("It dealt %d%% damage", attackPercent))

	damageState.State = state.Clone()

	states = append(states, damageState)

	if move.Meta.Drain > 0 {
		drainState := StateSnapshot{}
		var drainedHealth uint = 0

		cappedDamage := math.Min(float64(defPokemon.Hp.Value), float64(damage))

		drainPercent := float32(move.Meta.Drain) / float32(100)
		drainedHealth = uint(float32(cappedDamage) * drainPercent)

		attackPokemon.Heal(drainedHealth)

		drainedHealthPercent := int((float32(drainedHealth) / float32(attackPokemon.MaxHp)) * 100)

		log.Info().
			Float32("drainPercent", drainPercent).
			Uint("drainedHealth", drainedHealth).
			Int("drainedHealthPercent", drainedHealthPercent).
			Msg("Attack health drain")

		drainState.State = state.Clone()

		drainState.Messages = append(drainState.Messages, fmt.Sprintf("%s drained health from %s, healing %d%%", attackPokemon.Nickname, defPokemon.Nickname, drainedHealthPercent))
		states = append(states, drainState)
	}

	// Recoil
	if move.Meta.Drain < 0 {
		// Recoil will only be blocked by Rock Head (except for struggle)
		if move.Name == "struggle" || attackPokemon.Ability.Name != "rock-head" {
			recoilState := StateSnapshot{}
			recoilPercent := (float32(move.Meta.Drain) / 100)
			selfDamage := float32(attackPokemon.MaxHp) * recoilPercent
			attackPokemon.Damage(uint(selfDamage * -1))

			log.Info().
				Float32("recoilPercent", recoilPercent).
				Uint("selfDamage", uint(selfDamage)).
				Msg("Attack recoil")

			recoilState.State = state.Clone()

			recoilState.Messages = append(recoilState.Messages, fmt.Sprintf("%s took %d%% recoil damage", attackPokemon.Nickname, int(math.Abs(float64(move.Meta.Drain)))))
			states = append(states, recoilState)
		}
	}

	attackActionLogger().Debug().Float64("effectiveness", effectiveness).Msg("")

	effectivenessText := ""

	if effectiveness >= 2 {
		effectivenessText = "It was super effective!"
	} else if effectiveness <= 0.5 {
		effectivenessText = "It was not very effective"
	} else if effectiveness == 0 {
		effectivenessText = "It had no effect"
	}

	if effectivenessText != "" {
		states = append(states, NewMessageOnlySnapshot(effectivenessText))
	}

	return states
}

func ohkoHandler(state *GameState, attackPokemon *core.Pokemon, defPokemon *core.Pokemon) StateSnapshot {
	ohkoState := StateSnapshot{}

	if defPokemon.Level > attackPokemon.Level {
		ohkoState.Messages = append(ohkoState.Messages, "It failed!. Opponent's level is too high!")
		ohkoState.MessagesOnly = true
		return ohkoState
	}

	defPokemon.Damage(defPokemon.Hp.Value)

	randCheck := global.GokeRand.Float64()
	if randCheck < 0.01 {
		ohkoState.Messages = append(ohkoState.Messages, "%s took calamitous damage!", defPokemon.Nickname)
	} else {
		ohkoState.Messages = append(ohkoState.Messages, "It's a one-hit KO!")
	}

	ohkoState.State = state.Clone()

	return ohkoState
}

func ailmentHandler(state *GameState, defPokemon *core.Pokemon, move core.Move) StateSnapshot {
	ailment, ok := core.STATUS_NAME_MAP[move.Meta.Ailment.Name]
	if ok && defPokemon.Status == core.STATUS_NONE {
		ailmentCheck := global.GokeRand.IntN(100)
		ailmentChance := move.Meta.AilmentChance

		// in pokeapi speak, 0 here means the chance is 100% (at least as it relates to moves like toxic and poison-powder)
		// might have to fix edge-cases here
		if ailmentChance == 0 {
			ailmentChance = 100
		}

		if ailmentCheck < ailmentChance {
			attackActionLogger().
				Debug().
				Int("ailmentCheck", ailmentCheck).
				Int("AilmentChance", ailmentChance).
				Msg("Check succeeded")

			// Manual override of toxic so that it applies toxic and not poison
			if move.Name == "toxic" {
				ailment = core.STATUS_TOXIC
			}

			stateSnap := ApplyAilment(state, defPokemon, ailment)

			// Make sure the pokemon didn't avoid ailment with ability or such
			if defPokemon.Status != core.STATUS_NONE {
				attackActionLogger().Info().
					Msgf("%s was afflicted with ailment: %s:%d", defPokemon.Nickname, move.Meta.Ailment.Name, ailment)
			} else {
				log.Info().Msgf("pokemon removed ailment with ability: %s", defPokemon.Ability.Name)
			}

			return stateSnap
		} else {
			attackActionLogger().
				Debug().
				Int("ailmentCheck", ailmentCheck).
				Int("AilmentChance", ailmentChance).
				Msg("Check failed")
		}

	}

	effect, ok := core.EFFECT_NAME_MAP[move.Meta.Ailment.Name]
	if ok {
		effectChance := move.Meta.AilmentChance
		if effectChance == 0 {
			effectChance = 100
		}
		effectCheck := global.GokeRand.IntN(100)

		if effectCheck < effectChance {
			switch effect {
			case core.EFFECT_CONFUSION:
				// TODO: add message
				if defPokemon.Ability.Name != "own-tempo" {
					log.Info().Int("effectCheck", effectCheck).Int("effectChance", effectChance).Msg("confusion check passed")

					confusionDuration := global.GokeRand.IntN(3) + 2
					defPokemon.ConfusionCount = confusionDuration
					log.Info().Int("confusionCount", defPokemon.ConfusionCount).Msg("confusion applied")
				}
			}
		}
	}

	return StateSnapshot{
		State: state.Clone(),
	}
}

func healHandler(state *GameState, attackPokemon *core.Pokemon, move core.Move) StateSnapshot {
	healPercent := float64(move.Meta.Healing) / 100
	attackPokemon.HealPerc(healPercent)

	return NewStateSnapshot(state, fmt.Sprintf("%s healed by %d%%", attackPokemon.Nickname, move.Meta.Healing))
}

func forceSwitchHandler(state *GameState, defPlayer *Player) StateSnapshot {
	if defPlayer.GetActivePokemon().Ability.Name == "suction-cups" {
		return NewMessageOnlySnapshot(fmt.Sprintf("%s stays in with it's suction cups!", defPlayer.GetActivePokemon().Nickname))
	}

	// since the active pokemon is determined by the position
	// of the pokemon in the Player.Team slice, we have to store
	// that position which makes this clunky
	type enumPokemon struct {
		Index   int
		Pokemon core.Pokemon
	}

	enumeratedPkm := lo.Map(defPlayer.Team, func(pokemon core.Pokemon, i int) enumPokemon {
		return enumPokemon{
			Index:   i,
			Pokemon: pokemon,
		}
	})

	alivePokemon := lo.Filter(enumeratedPkm, func(e enumPokemon, _ int) bool {
		return e.Pokemon.Alive() && e.Index != defPlayer.ActivePokeIndex
	})

	if len(alivePokemon) == 0 {
		return NewMessageOnlySnapshot(fmt.Sprintf("%s has no Pokemon left to switch!", defPlayer.Name))
	}

	choiceIndex := global.GokeRand.IntN(len(alivePokemon))

	ogPokemonName := defPlayer.GetActivePokemon().Nickname

	defPlayer.ActivePokeIndex = alivePokemon[choiceIndex].Index
	defPlayer.GetActivePokemon().SwitchedInThisTurn = true
	defPlayer.GetActivePokemon().CanAttackThisTurn = false

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s was forced to switch out", ogPokemonName)},
	}
}

func (a AttackAction) GetCtx() ActionCtx {
	return a.Ctx
}
