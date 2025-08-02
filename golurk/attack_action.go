package golurk

import (
	"fmt"
	"math"

	"github.com/go-logr/logr"
	"github.com/samber/lo"
)

var attackEventLogger = func() logr.Logger {
	return internalLogger.WithName("attack_event")
}

var struggleMove = Move{
	Accuracy:    100,
	DamageClass: "physical",
	Meta: MoveMeta{
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

func (a AttackAction) UpdateState(state GameState) []StateEvent {
	return []StateEvent{AttackEvent{AttackerID: a.Ctx.PlayerID, MoveID: a.AttackerMove}}
}

func damageMoveHandler(state GameState, attackPokemon Pokemon, attIndex int, defPokemon Pokemon, defIndex int, move Move) []StateEvent {
	events := make([]StateEvent, 0)
	crit := false

	rng := state.CreateRng()

	if rng.Float32() < attackPokemon.CritChance() {
		crit = true
		attackEventLogger().Info("Attack Crit!", "chance", attackPokemon.CritChance())
	}

	effectiveness := defPokemon.DefenseEffectiveness(GetAttackTypeMapping(move.Type))

	if crit && (defPokemon.Ability.Name == "battle-armor" || defPokemon.Ability.Name == "shell-armor") {
		events = append(events, AbilityActivationEvent{ActivatorInt: defIndex, AbilityName: defPokemon.Ability.Name})
		crit = false
	}

	damage := Damage(attackPokemon, defPokemon, move, crit, state.Weather, rng)

	if defPokemon.Ability.Name == "sturdy" {
		if damage >= defPokemon.Hp.Value && defPokemon.Hp.Value == defPokemon.MaxHp {
			// set the defending pokemon's hp to 1
			damage = defPokemon.MaxHp - 1
			events = append(events,
				SimpleAbilityActivationEvent(&state, defIndex),
				NewFmtMessageEvent("%s held on!", defPokemon.Nickname),
			)
		}
	}

	if defPokemon.Ability.Name == "wonder-guard" {
		if effectiveness < 2 {
			events = append(events,
				SimpleAbilityActivationEvent(&state, defIndex),
				NewFmtMessageEvent("%s does not take any damage!", defPokemon.Nickname),
			)

			return events
		}
	}

	if defPokemon.Ability.Name == "lightning-rod" && move.Type == TYPENAME_ELECTRIC {
		events = append(events, SimpleAbilityActivationEvent(&state, defIndex))
	}

	events = append(events, DamageEvent{PlayerIndex: defIndex, Damage: damage, Crit: crit})

	attackEventLogger().Info("Attack Event!", "attacker", attackPokemon.Nickname, "defender", defPokemon.Nickname, "damage", damage)

	if move.Meta.Drain > 0 {
		var drainedHealth uint = 0

		cappedDamage := math.Min(float64(defPokemon.Hp.Value), float64(damage))

		drainPercent := float32(move.Meta.Drain) / float32(100)
		drainedHealth = uint(float32(cappedDamage) * drainPercent)

		events = append(events, HealEvent{Heal: drainedHealth, PlayerIndex: attIndex})

		drainedHealthPercent := int((float32(drainedHealth) / float32(attackPokemon.MaxHp)) * 100)

		attackEventLogger().Info("Drain", "percent", drainPercent, "drained_health", drainedHealth, "drained_health_percent", drainedHealthPercent)
	}

	// Recoil
	if move.Meta.Drain < 0 {
		// Recoil will only be blocked by Rock Head (except for struggle)
		if move.Name == "struggle" || attackPokemon.Ability.Name != "rock-head" {
			recoilPercent := (float32(move.Meta.Drain) / 100)
			selfDamage := float32(attackPokemon.MaxHp) * recoilPercent

			events = append(events, NewFmtMessageEvent("%s took %d%% recoil damage", attackPokemon.Nickname, int(math.Abs(float64(move.Meta.Drain)))))
			// flip sign here because recoil is considered negative Drain healing in pokeapi
			events = append(events, DamageEvent{Damage: uint(selfDamage * -1), PlayerIndex: attIndex, SupressMessage: true})

			attackEventLogger().Info("Recoil", "recoil_percent", recoilPercent, "self_damage", selfDamage)
		}
	}

	effectivenessText := ""

	if effectiveness >= 2 {
		effectivenessText = "It was super effective!"
	} else if effectiveness <= 0.5 {
		effectivenessText = "It was not very effective"
	} else if effectiveness == 0 {
		effectivenessText = "It had no effect"
	}

	if effectivenessText != "" {
		events = append(events, NewMessageEvent(effectivenessText))
	}

	if defPokemon.Ability.Name == "color-change" && move.Name != "struggle" {
		moveType := GetAttackTypeMapping(move.Type)
		if !defPokemon.HasType(moveType) {
			events = append(events, TypeChangeEvent{ChangerInt: defIndex, PokemonType: *GetAttackTypeMapping(move.Type)})
		}
	}

	return events
}

func ohkoHandler(state *GameState, attackPokemon Pokemon, defPokemon Pokemon, defIndex int) []StateEvent {
	if defPokemon.Level > attackPokemon.Level {
		return []StateEvent{NewMessageEvent("It failed!. Opponent's level is too high!")}
	}

	rng := state.CreateRng()

	events := make([]StateEvent, 0)
	events = append(events, DamageEvent{PlayerIndex: defIndex, Damage: defPokemon.Hp.Value})

	randCheck := rng.Float64()
	if randCheck < 0.01 {
		events = append(events, NewFmtMessageEvent("%s took calamitous damage!", defPokemon.Nickname))
	} else {
		events = append(events, NewMessageEvent("It's a one-hit KO!"))
	}

	return events
}

func ailmentHandler(state GameState, defPokemon Pokemon, defIndex int, move Move) []StateEvent {
	ailment, ok := STATUS_NAME_MAP[move.Meta.Ailment.Name]
	rng := state.CreateRng()
	if ok && defPokemon.Status == STATUS_NONE {
		ailmentCheck := rng.IntN(100)
		ailmentChance := move.Meta.AilmentChance

		// in pokeapi speak, 0 here means the chance is 100% (at least as it relates to moves like toxic and poison-powder)
		// might have to fix edge-cases here
		if ailmentChance == 0 {
			ailmentChance = 100
		}

		if ailmentCheck < ailmentChance {
			attackEventLogger().Info("Ailment check succeeded!", "chance", ailmentChance, "ailment_check", ailmentCheck)

			// Manual override of toxic so that it applies toxic and not poison
			if move.Name == "toxic" {
				ailment = STATUS_TOXIC
			}

			event := AilmentEvent{PlayerIndex: defIndex, Ailment: ailment}

			// Make sure the pokemon didn't avoid ailment with ability or such
			if defPokemon.Status != STATUS_NONE {
				attackEventLogger().Info("Pokemon afflicted with ailment", "pokemon_name", defPokemon.Nickname, "ailment_name", move.Meta.Ailment.Name, "ailment_id", ailment)
			} else {
				attackEventLogger().Info("Pokemon removed ailment with ability", "ability_name", defPokemon.Ability.Name, "ailment_id", ailment)
			}

			return []StateEvent{event}
		} else {
			attackEventLogger().Info("Ailment check failed.", "ailment_chance", ailmentChance, "ailment_check", ailmentCheck)
		}

	}

	effect, ok := EFFECT_NAME_MAP[move.Meta.Ailment.Name]
	if ok {
		effectChance := move.Meta.AilmentChance
		if effectChance == 0 {
			effectChance = 100
		}
		effectCheck := rng.IntN(100)

		if effectCheck < effectChance {
			switch effect {
			case EFFECT_CONFUSION:
				// TODO: add message
				if defPokemon.Ability.Name != "own-tempo" {
					attackEventLogger().Info("Confusion check passed.", "effect_chance", effectChance, "effect_check", effectCheck)

					return []StateEvent{ApplyConfusionEvent{PlayerIndex: defIndex}}
				}
			}
		}
	}

	return nil
}

func healHandler(state *GameState, pokemonIndex int, move Move) StateEvent {
	healPercent := float64(move.Meta.Healing) / 100
	return HealPercEvent{PlayerIndex: pokemonIndex, HealPerc: healPercent}
}

func forceSwitchHandler(state *GameState, defPlayer *Player, defIndex int) []StateEvent {
	defPokemon := defPlayer.GetActivePokemon()
	if defPokemon.Ability.Name == "suction-cups" {
		return []StateEvent{
			SimpleAbilityActivationEvent(state, defIndex),
			NewFmtMessageEvent("%s cannot be forced out!", defPokemon.Nickname),
		}
	}

	// since the active pokemon is determined by the position
	// of the pokemon in the Player.Team slice, we have to store
	// that position which makes this clunky
	type enumPokemon struct {
		Index   int
		Pokemon Pokemon
	}

	enumeratedPkm := lo.Map(defPlayer.Team, func(pokemon Pokemon, i int) enumPokemon {
		return enumPokemon{
			Index:   i,
			Pokemon: pokemon,
		}
	})

	alivePokemon := lo.Filter(enumeratedPkm, func(e enumPokemon, _ int) bool {
		return e.Pokemon.Alive() && e.Index != defPlayer.ActivePokeIndex
	})

	if len(alivePokemon) == 0 {
		return []StateEvent{NewFmtMessageEvent(fmt.Sprintf("%s has no Pokemon left to switch in!", defPlayer.Name))}
	}

	rng := state.CreateRng()

	choiceIndex := rng.IntN(len(alivePokemon))

	return []StateEvent{
		SwitchEvent{PlayerIndex: defIndex, SwitchIndex: alivePokemon[choiceIndex].Index},
	}
}

func (a AttackAction) GetCtx() ActionCtx {
	return a.Ctx
}
