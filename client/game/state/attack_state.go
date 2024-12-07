package state

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func (a *AttackAction) UpdateState(state GameState) StateUpdate {
	attacker := state.GetPlayer(a.ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	move := attackPokemon.Moves[a.AttackerMove]

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s used %s", attackPokemon.Nickname, move.Name))

	accuracyCheck := rand.Intn(100)
	accuracy := move.Accuracy
	if accuracy == 0 {
		accuracy = 100
	}

	if accuracyCheck < accuracy {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check passed")
		// TODO: handle these categories
		// - net-good-stats
		// - swagger
		// - damage+lower
		// - damage+raise
		// - ohko
		// - force-switch
		// - unique

		switch move.Meta.Category.Name {
		case "damage", "damage+heal":
			messages = append(messages, damageMoveHandler(attackPokemon, defPokemon, move)...)
		case "ailment":
			ailmentHandler(defPokemon, move)
		case "damage+ailment":
			messages = append(messages, damageMoveHandler(attackPokemon, defPokemon, move)...)
			ailmentHandler(defPokemon, move)
		case "heal":
			messages = append(messages, healHandler(attackPokemon, move)...)
		default:
			attackActionLogger().Warn().Msgf("Move, %s (%s category), has no handler!!!", move.Name, move.Meta.Category.Name)
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check failed")
		messages = append(messages, "It missed!")
	}

	return StateUpdate{
		State:    state,
		Messages: messages,
	}
}

func damageMoveHandler(attackPokemon *game.Pokemon, defPokemon *game.Pokemon, move *game.Move) []string {
	messages := make([]string, 0)

	damage := game.Damage(*attackPokemon, *defPokemon, move)
	log.Info().Msgf("%s attacked %s, dealing %d damage", attackPokemon.Nickname, defPokemon.Nickname, damage)

	attackActionLogger().Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	attackPercent := uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	defPokemon.Damage(damage)
	var drainedHealth uint = 0

	// TODO: need to make this a separate "update" as well so it changes visually
	if move.Meta.Drain > 0 {
		drainPercent := float32(move.Meta.Drain) / float32(100)
		drainedHealth = uint(float32(damage) * drainPercent)

		attackPokemon.Heal(drainedHealth)

		drainedHealthPercent := int((float32(drainedHealth) / float32(attackPokemon.MaxHp)) * 100)

		log.Info().
			Float32("drainPercent", drainPercent).
			Uint("drainedHealth", drainedHealth).
			Int("drainedHealthPercent", drainedHealthPercent).
			Msg("Attack health drain")

		messages = append(messages, fmt.Sprintf("%s drained health from %s, healing %d%%", attackPokemon.Nickname, defPokemon.Nickname, drainedHealthPercent))
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

	messages = append(messages, fmt.Sprintf("It dealt %d%% damage", attackPercent))

	if effectivenessText != "" {
		messages = append(messages, effectivenessText)
	}

	return messages
}

func ailmentHandler(defPokemon *game.Pokemon, move *game.Move) {
	// TODO: Setup state updates so that this can be in its own separate update
	// (probably make update functions return []StateUpdate)
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
}

func healHandler(attackPokemon *game.Pokemon, move *game.Move) []string {
	messages := make([]string, 0)

	healPercent := float32(move.Meta.Healing) / 100
	healAmount := float32(attackPokemon.MaxHp) * healPercent

	attackPokemon.Heal(uint(healAmount))

	messages = append(messages, fmt.Sprintf("%s healed by %d%%", attackPokemon.Nickname, move.Meta.Healing))

	return messages
}

func (a AttackAction) Ctx() ActionCtx {
	return a.ctx
}
