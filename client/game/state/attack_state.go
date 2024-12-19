package state

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var attackActionLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "attack-action").Logger()
	return &logger
}

type AttackAction struct {
	ctx ActionCtx

	AttackerMove int
}

func NewAttackAction(attacker int, attackMove int) *AttackAction {
	return &AttackAction{
		ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

func (a *AttackAction) UpdateState(state GameState) []StateUpdate {
	attacker := state.GetPlayer(a.ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	move := attackPokemon.Moves[a.AttackerMove]

	states := make([]StateUpdate, 0)

	// "hack" to show this messages first
	useMoveState := StateUpdate{}
	useMoveState.State = state.Clone()
	useMoveState.Messages = append(useMoveState.Messages, fmt.Sprintf("%s used %s", attackPokemon.Nickname, move.Name))

	states = append(states, useMoveState)

	accuracyCheck := rand.Intn(100)
	accuracy := move.Accuracy
	if accuracy == 0 {
		accuracy = 100
	}

	if accuracyCheck < accuracy {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check passed")
		// TODO: handle these categories
		// - swagger
		// - ohko
		// - force-switch
		// - unique

		switch move.Meta.Category.Name {
		case "damage", "damage+heal":
			states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
		case "ailment":
			ailmentHandler(&state, defPokemon, move)
		case "damage+ailment":
			states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
			ailmentHandler(&state, defPokemon, move)
		case "net-good-stats":
			lo.ForEach(move.StatChanges, func(statChange game.StatChange, _ int) {
				// since its "net-good-stats", the stat change always has to benefit the user
				affectedPokemon := attackPokemon
				if statChange.Change < 0 {
					affectedPokemon = defPokemon
				}

				states = append(states, statChangeHandler(&state, affectedPokemon, statChange, move.Meta.StatChance))
			})
		// Damages and then CHANGES the targets stats
		case "damage+lower":
			states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
			lo.ForEach(move.StatChanges, func(statChange game.StatChange, _ int) {
				states = append(states, statChangeHandler(&state, defPokemon, statChange, move.Meta.StatChance))
			})
		// Damages and then CHANGES the user's stats
		// this is different from what pokeapi says (raises instead of changes)
		// and this is important because moves like draco-meteor and overheat
		// lower the user's stats but are in this category
		case "damage+raise":
			states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
			lo.ForEach(move.StatChanges, func(statChange game.StatChange, _ int) {
				states = append(states, statChangeHandler(&state, attackPokemon, statChange, move.Meta.StatChance))
			})
		case "heal":
			states = append(states, healHandler(&state, attackPokemon, move))
		default:
			attackActionLogger().Warn().Msgf("Move, %s (%s category), has no handler!!!", move.Name, move.Meta.Category.Name)
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check failed")
		states = append(states, StateUpdate{
			State:    state.Clone(),
			Messages: []string{"It Missed!"},
		})
	}

	finalState := StateUpdate{
		State: state.Clone(),
	}
	states = append(states, finalState)

	return states
}

func damageMoveHandler(state *GameState, attackPokemon *game.Pokemon, defPokemon *game.Pokemon, move *game.Move) []StateUpdate {
	states := make([]StateUpdate, 0)

	damage := game.Damage(*attackPokemon, *defPokemon, move)
	log.Info().Msgf("%s attacked %s, dealing %d damage", attackPokemon.Nickname, defPokemon.Nickname, damage)

	attackActionLogger().Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	attackPercent := uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	defPokemon.Damage(damage)
	var drainedHealth uint = 0

	// TODO: need to make this a separate "update" as well so it changes visually
	if move.Meta.Drain > 0 {
		drainState := StateUpdate{}

		drainPercent := float32(move.Meta.Drain) / float32(100)
		drainedHealth = uint(float32(damage) * drainPercent)

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

	effectiveness := defPokemon.Base.DefenseEffectiveness(game.GetAttackTypeMapping(move.Type))

	attackActionLogger().Debug().Float32("effectiveness", effectiveness).Msg("")

	effectivenessText := ""

	if effectiveness >= 2 {
		effectivenessText = "It was super effective!"
	} else if effectiveness <= 0.5 {
		effectivenessText = "It was not very effective"
	} else if effectiveness == 0 {
		effectivenessText = "It had no effect"
	}

	damageState := StateUpdate{}
	damageState.State = state.Clone()

	damageState.Messages = append(damageState.Messages, fmt.Sprintf("It dealt %d%% damage", attackPercent))

	if effectivenessText != "" {
		damageState.Messages = append(damageState.Messages, effectivenessText)
	}

	states = append(states, damageState)

	return states
}

func ailmentHandler(state *GameState, defPokemon *game.Pokemon, move *game.Move) StateUpdate {
	ailment, ok := game.STATUS_NAME_MAP[move.Meta.Ailment.Name]
	if ok && defPokemon.Status == game.STATUS_NONE {
		ailmentCheck := rand.Intn(100)
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

			defPokemon.Status = ailment

			// Manual override of toxic so that it applies toxic and not poison
			if move.Name == "toxic" {
				defPokemon.Status = game.STATUS_TOXIC
			}

			// Post-Ailment initialization
			switch defPokemon.Status {
			// Set how many turns the pokemon is asleep for
			case game.STATUS_SLEEP:
				randTime := rand.Intn(2) + 1
				defPokemon.SleepCount = randTime
				attackActionLogger().Debug().Msgf("%s is now asleep for %d turns", defPokemon.Nickname, defPokemon.SleepCount)
			case game.STATUS_TOXIC:
				defPokemon.ToxicCount = 1
			}

			attackActionLogger().Info().
				Msgf("%s was afflicted with ailment: %s:%d", defPokemon.Nickname, move.Meta.Ailment.Name, ailment)
		} else {
			attackActionLogger().
				Debug().
				Int("ailmentCheck", ailmentCheck).
				Int("AilmentChance", ailmentChance).
				Msg("Check failed")
		}
	}

	effect, ok := game.EFFECT_NAME_MAP[move.Meta.Ailment.Name]
	if ok {
		effectChance := move.Meta.AilmentChance
		if effectChance == 0 {
			effectChance = 100
		}
		effectCheck := rand.Intn(100)

		if effectCheck < effectChance {
			switch effect {
			case game.EFFECT_CONFUSION:
				log.Info().Int("effectCheck", effectCheck).Int("effectChance", effectChance).Msg("confusion check passed")

				confusionDuration := rand.Intn(3) + 2
				defPokemon.ConfusionCount = confusionDuration
				log.Info().Int("confusionCount", defPokemon.ConfusionCount).Msg("confusion applied")
			}
		}
	}

	return StateUpdate{
		State: state.Clone(),
	}
}

func healHandler(state *GameState, attackPokemon *game.Pokemon, move *game.Move) StateUpdate {
	healState := StateUpdate{}

	healPercent := float32(move.Meta.Healing) / 100
	healAmount := float32(attackPokemon.MaxHp) * healPercent

	attackPokemon.Heal(uint(healAmount))

	healState.State = state.Clone()

	healState.Messages = append(healState.Messages, fmt.Sprintf("%s healed by %d%%", attackPokemon.Nickname, move.Meta.Healing))

	return healState
}

func statChangeHandler(state *GameState, pokemon *game.Pokemon, statChange game.StatChange, statChance int) StateUpdate {
	statCheck := rand.Intn(100)
	if statChance == 0 {
		statChance = 100
	}

	statChangeState := StateUpdate{}

	if statCheck < statChance {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did pass")
		statChangeState.Messages = append(statChangeState.Messages, changeStat(pokemon, statChange.Stat.Name, statChange.Change)...)
	} else {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did not pass")
	}

	statChangeState.State = state.Clone()

	return statChangeState
}

func changeStat(pokemon *game.Pokemon, statName string, change int) []string {
	messages := make([]string, 0)

	absChange := int(math.Abs(float64(change)))
	if change > 0 {
		messages = append(messages, fmt.Sprintf("%s's %s increased by %d stages!", pokemon.Nickname, statName, absChange))
	} else {
		messages = append(messages, fmt.Sprintf("%s's %s decreased by %d stages!", pokemon.Nickname, statName, absChange))
	}

	// sorry
	switch statName {
	case "attack":
		pokemon.Attack.ChangeStat(change)
	case "defense":
		pokemon.Def.ChangeStat(change)
	case "special-attack":
		pokemon.SpAttack.ChangeStat(change)
	case "special-defense":
		pokemon.SpDef.ChangeStat(change)
	case "speed":
		pokemon.RawSpeed.ChangeStat(change)
	}

	return messages
}

func (a AttackAction) Ctx() ActionCtx {
	return a.ctx
}
