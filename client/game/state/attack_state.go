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

var struggleMove = game.Move{
	Accuracy:    100,
	DamageClass: "physical",
	Meta: &game.MoveMeta{
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
	ctx ActionCtx

	AttackerMove int
}

func NewAttackAction(attacker int, attackMove int) *AttackAction {
	return &AttackAction{
		ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

func (a *AttackAction) UpdateState(state GameState) []StateSnapshot {
	attacker := state.GetPlayer(a.ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	var move *game.Move
	var moveVars game.BattleMove
	var pp uint

	if a.AttackerMove == -1 {
		move = &struggleMove
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

	accuracyCheck := rand.Intn(100)
	accuracy := move.Accuracy
	if accuracy == 0 {
		accuracy = 100
	}

	if accuracyCheck < accuracy && pp > 0 {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check passed")
		// TODO: handle these categories
		// - swagger
		// - unique

		defImmune := false

		// TODO: This doesn't activate through protect!
		if move.Type == game.TYPENAME_ELECTRIC && defPokemon.Ability.Name == "volt-absorb" {
			states = append(states, NewMessageOnlySnapshot(
				fmt.Sprintf("%s activated volt absorb!", defPokemon.Nickname),
				fmt.Sprintf("%s healed 25%%!", defPokemon.Nickname)),
			)

			defPokemon.HealPerc(.25)
			defImmune = true
		}

		// TODO: This doesn't activate through protect!
		if move.Type == game.TYPENAME_WATER && defPokemon.Ability.Name == "water-absorb" {
			states = append(states, NewMessageOnlySnapshot(
				fmt.Sprintf("%s activated Water Absorb!", defPokemon.Nickname),
				fmt.Sprintf("%s healed 25%%!", defPokemon.Nickname)),
			)

			defPokemon.HealPerc(.25)
			defImmune = true
		}

		if !defImmune {
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

					states = append(states, StatChangeHandler(&state, affectedPokemon, statChange, move.Meta.StatChance))
				})
			// Damages and then CHANGES the targets stats
			case "damage+lower":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
				lo.ForEach(move.StatChanges, func(statChange game.StatChange, _ int) {
					states = append(states, StatChangeHandler(&state, defPokemon, statChange, move.Meta.StatChance))
				})
			// Damages and then CHANGES the user's stats
			// this is different from what pokeapi says (raises instead of changes)
			// and this is important because moves like draco-meteor and overheat
			// lower the user's stats but are in this category
			case "damage+raise":
				states = append(states, damageMoveHandler(&state, attackPokemon, defPokemon, move)...)
				lo.ForEach(move.StatChanges, func(statChange game.StatChange, _ int) {
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

		if rand.Intn(100) < move.Meta.FlinchChance {
			states = append(states, FlinchHandler(&state, defPokemon))
		}

		if a.AttackerMove != -1 {
			if pp == 0 {
				attackPokemon.InGameMoveInfo[a.AttackerMove].PP = 0
			} else {
				attackPokemon.InGameMoveInfo[a.AttackerMove].PP = pp - 1
			}
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check failed")
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

func damageMoveHandler(state *GameState, attackPokemon *game.Pokemon, defPokemon *game.Pokemon, move *game.Move) []StateSnapshot {
	states := make([]StateSnapshot, 0)

	crit := false

	if rand.Float32() < attackPokemon.CritChance() {
		crit = true
		log.Info().Float32("chance", attackPokemon.CritChance()).Msg("Attack crit!")
	}

	damage := game.Damage(*attackPokemon, *defPokemon, move, crit)

	if defPokemon.Ability.Name == "sturdy" {
		if damage >= defPokemon.Hp.Value && defPokemon.Hp.Value == defPokemon.MaxHp {
			// set the defending pokemon's hp to 1
			damage = defPokemon.MaxHp - 1
			states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s activated sturdy and held on!", defPokemon.Nickname)))
		}
	}

	log.Info().Msgf("%s attacked %s, dealing %d damage", attackPokemon.Nickname, defPokemon.Nickname, damage)

	attackActionLogger().Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	attackPercent := uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	damageState := StateSnapshot{}

	if crit {
		damageState.Messages = append(damageState.Messages, fmt.Sprintf("%s critically hit!", attackPokemon.Nickname))
	}
	damageState.Messages = append(damageState.Messages, fmt.Sprintf("It dealt %d%% damage", attackPercent))

	defPokemon.Damage(damage)
	var drainedHealth uint = 0

	damageState.State = state.Clone()

	states = append(states, damageState)

	if move.Meta.Drain > 0 {
		drainState := StateSnapshot{}

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

	if move.Meta.Drain < 0 {
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

	if effectivenessText != "" {
		states = append(states, NewMessageOnlySnapshot(effectivenessText))
	}

	return states
}

func ohkoHandler(state *GameState, attackPokemon *game.Pokemon, defPokemon *game.Pokemon) StateSnapshot {
	ohkoState := StateSnapshot{}

	if defPokemon.Level > attackPokemon.Level {
		ohkoState.Messages = append(ohkoState.Messages, "It failed!. Opponent's level is too high!")
		ohkoState.MessagesOnly = true
		return ohkoState
	}

	defPokemon.Damage(defPokemon.Hp.Value)

	randCheck := rand.Float64()
	if randCheck < 0.01 {
		ohkoState.Messages = append(ohkoState.Messages, "%s took calamitous damage!", defPokemon.Nickname)
	} else {
		ohkoState.Messages = append(ohkoState.Messages, "It's a one-hit KO!")
	}

	ohkoState.State = state.Clone()

	return ohkoState
}

func ailmentHandler(state *GameState, defPokemon *game.Pokemon, move *game.Move) StateSnapshot {
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

	return StateSnapshot{
		State: state.Clone(),
	}
}

func healHandler(state *GameState, attackPokemon *game.Pokemon, move *game.Move) StateSnapshot {
	healState := StateSnapshot{}

	healPercent := float32(move.Meta.Healing) / 100
	healAmount := float32(attackPokemon.MaxHp) * healPercent

	attackPokemon.Heal(uint(healAmount))

	healState.State = state.Clone()

	healState.Messages = append(healState.Messages, fmt.Sprintf("%s healed by %d%%", attackPokemon.Nickname, move.Meta.Healing))

	return healState
}

func forceSwitchHandler(state *GameState, defPlayer *Player) StateSnapshot {
	// since the active pokemon is determined by the position
	// of the pokemon in the Player.Team slice, we have to store
	// that position which makes this clunky
	type enumPokemon struct {
		Index   int
		Pokemon game.Pokemon
	}

	enumeratedPkm := lo.Map(defPlayer.Team, func(pokemon game.Pokemon, i int) enumPokemon {
		return enumPokemon{
			Index:   i,
			Pokemon: pokemon,
		}
	})

	alivePokemon := lo.Filter(enumeratedPkm, func(e enumPokemon, _ int) bool {
		return e.Pokemon.Alive() && e.Index != defPlayer.ActivePokeIndex
	})

	choiceIndex := rand.Intn(len(alivePokemon))

	ogPokemonName := defPlayer.GetActivePokemon().Nickname

	defPlayer.ActivePokeIndex = alivePokemon[choiceIndex].Index
	defPlayer.GetActivePokemon().SwitchedInThisTurn = true
	defPlayer.GetActivePokemon().CanAttackThisTurn = false

	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s was forced to switch out", ogPokemonName)},
	}
}

func (a AttackAction) Ctx() ActionCtx {
	return a.ctx
}
