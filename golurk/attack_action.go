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
			Id   int    `json:"id"`
			Name string `json:"name"`
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

type attackHandlerContext struct {
	gameState GameState
	attacker  int
	defender  int
	move      Move
}

func newAttackHandlerContext(gameState GameState, attacker int, defender int, move Move) attackHandlerContext {
	return attackHandlerContext{gameState, attacker, defender, move}
}

func (ctx attackHandlerContext) attackPokemon() Pokemon {
	attackPlayer := ctx.gameState.GetPlayer(ctx.attacker)
	return *attackPlayer.GetActivePokemon()
}

func (ctx attackHandlerContext) defPokemon() Pokemon {
	defPlayer := ctx.gameState.GetPlayer(ctx.defender)
	return *defPlayer.GetActivePokemon()
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

func damageMoveHandler(ctx attackHandlerContext) []StateEvent {
	events := make([]StateEvent, 0)
	crit := false

	attackPokemon := ctx.attackPokemon()
	defPokemon := ctx.defPokemon()

	rng := ctx.gameState.CreateRng()

	if rng.Float32() < attackPokemon.CritChance() {
		crit = true
		attackEventLogger().Info("Attack Crit!", "chance", attackPokemon.CritChance())
	}

	effectiveness := defPokemon.DefenseEffectiveness(GetAttackTypeMapping(ctx.move.Type))

	if crit && (defPokemon.Ability.Name == "battle-armor" || defPokemon.Ability.Name == "shell-armor") {
		events = append(events, AbilityActivationEvent{ActivatorInt: ctx.defender, AbilityName: defPokemon.Ability.Name})
		crit = false
	}

	damage := Damage(attackPokemon, defPokemon, ctx.move, crit, ctx.gameState.Weather, rng)

	if defPokemon.Ability.Name == "sturdy" {
		if damage >= defPokemon.Hp.Value && defPokemon.Hp.Value == defPokemon.MaxHp {
			// set the defending pokemon's hp to 1
			damage = defPokemon.MaxHp - 1
			events = append(events,
				SimpleAbilityActivationEvent(&ctx.gameState, ctx.defender),
				NewFmtMessageEvent("%s held on!", defPokemon.Name()),
			)
		}
	}

	if defPokemon.Ability.Name == "wonder-guard" {
		if effectiveness < 2 {
			events = append(events,
				SimpleAbilityActivationEvent(&ctx.gameState, ctx.defender),
				NewFmtMessageEvent("%s does not take any damage!", defPokemon.Name()),
			)

			return events
		}
	}

	if defPokemon.Ability.Name == "lightning-rod" && ctx.move.Type == TYPENAME_ELECTRIC {
		events = append(events, SimpleAbilityActivationEvent(&ctx.gameState, ctx.defender))
	}

	events = append(events, DamageEvent{PlayerIndex: ctx.defender, Damage: damage, Crit: crit})

	attackEventLogger().Info("Attack Event!", "attacker", attackPokemon.Name(), "defender", defPokemon.Name(), "damage", damage)

	if ctx.move.Meta.Drain > 0 {
		var drainedHealth uint = 0

		cappedDamage := math.Min(float64(defPokemon.Hp.Value), float64(damage))

		drainPercent := float32(ctx.move.Meta.Drain) / float32(100)
		drainedHealth = uint(float32(cappedDamage) * drainPercent)

		events = append(events, HealEvent{Heal: drainedHealth, PlayerIndex: ctx.attacker})

		drainedHealthPercent := int((float32(drainedHealth) / float32(attackPokemon.MaxHp)) * 100)

		attackEventLogger().Info("Drain", "percent", drainPercent, "drained_health", drainedHealth, "drained_health_percent", drainedHealthPercent)
	}

	// Recoil
	if ctx.move.Meta.Drain < 0 {
		// Recoil will only be blocked by Rock Head (except for struggle)
		if ctx.move.Name == "struggle" || attackPokemon.Ability.Name != "rock-head" {
			recoilPercent := (float32(ctx.move.Meta.Drain) / 100)
			selfDamage := float32(attackPokemon.MaxHp) * recoilPercent

			events = append(events, NewFmtMessageEvent("%s took %d%% recoil damage", attackPokemon.Name(), int(math.Abs(float64(ctx.move.Meta.Drain)))))
			// flip sign here because recoil is considered negative Drain healing in pokeapi
			events = append(events, DamageEvent{Damage: uint(selfDamage * -1), PlayerIndex: ctx.attacker, SupressMessage: true})

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

	if defPokemon.Ability.Name == "color-change" && ctx.move.Name != "struggle" {
		moveType := GetAttackTypeMapping(ctx.move.Type)
		if !defPokemon.HasType(moveType) {
			events = append(events, TypeChangeEvent{ChangerInt: ctx.defender, PokemonType: *GetAttackTypeMapping(ctx.move.Type)})
		}
	}

	return events
}

func ohkoHandler(ctx attackHandlerContext) []StateEvent {
	attackPokemon := ctx.attackPokemon()
	defPokemon := ctx.defPokemon()
	if defPokemon.Level > attackPokemon.Level {
		return []StateEvent{NewMessageEvent("It failed!. Opponent's level is too high!")}
	}

	rng := ctx.gameState.CreateRng()

	events := make([]StateEvent, 0)
	events = append(events, DamageEvent{PlayerIndex: ctx.defender, Damage: defPokemon.Hp.Value})

	randCheck := rng.Float64()
	if randCheck < 0.01 {
		events = append(events, NewFmtMessageEvent("%s took calamitous damage!", defPokemon.Name()))
	} else {
		events = append(events, NewMessageEvent("It's a one-hit KO!"))
	}

	return events
}

func ailmentHandler(ctx attackHandlerContext) []StateEvent {
	defPokemon := ctx.defPokemon()
	attackPokemon := ctx.attackPokemon()

	ailment, ok := STATUS_NAME_MAP[ctx.move.Meta.Ailment.Name]
	rng := ctx.gameState.CreateRng()
	if ok && defPokemon.Status == STATUS_NONE {
		ailmentCheck := rng.IntN(100)
		ailmentChance := ctx.move.Meta.AilmentChance

		// in pokeapi speak, 0 here means the chance is 100% (at least as it relates to moves like toxic and poison-powder)
		// might have to fix edge-cases here
		if ailmentChance == 0 {
			ailmentChance = 100
		}

		if ailmentCheck < ailmentChance {
			attackEventLogger().Info("Ailment check succeeded!", "chance", ailmentChance, "ailment_check", ailmentCheck)

			// Manual override of toxic so that it applies toxic and not poison
			if ctx.move.Name == "toxic" {
				ailment = STATUS_TOXIC
			}

			event := AilmentEvent{PlayerIndex: ctx.defender, Ailment: ailment}

			// Make sure the pokemon didn't avoid ailment with ability or such
			if defPokemon.Status != STATUS_NONE {
				attackEventLogger().Info("Pokemon afflicted with ailment", "pokemon_name", defPokemon.Name(), "ailment_name", ctx.move.Meta.Ailment.Name, "ailment_id", ailment)
			} else {
				attackEventLogger().Info("Pokemon removed ailment with ability", "ability_name", defPokemon.Ability.Name, "ailment_id", ailment)
			}

			return []StateEvent{event}
		} else {
			attackEventLogger().Info("Ailment check failed.", "ailment_chance", ailmentChance, "ailment_check", ailmentCheck)
		}

	}

	effect, ok := EFFECT_NAME_MAP[ctx.move.Meta.Ailment.Name]
	if ok {
		effectChance := ctx.move.Meta.AilmentChance
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

					return []StateEvent{ApplyConfusionEvent{PlayerIndex: ctx.defender}}
				}
			case EFFECT_INFATUATION:
				if defPokemon.Gender != attackPokemon.Gender && defPokemon.Gender != "unknown" && attackPokemon.Gender != "unknown" {
					return []StateEvent{ApplyInfatuationEvent{PlayerIndex: ctx.defender}}
				}
			}
		}
	}

	return nil
}

// creates a heal event for attacker
func healHandler(ctx attackHandlerContext) StateEvent {
	healPercent := float64(ctx.move.Meta.Healing) / 100
	return HealPercEvent{PlayerIndex: ctx.attacker, HealPerc: healPercent}
}

func forceSwitchHandler(ctx attackHandlerContext) []StateEvent {
	defPokemon := ctx.defPokemon()
	defPlayer := ctx.gameState.GetPlayer(ctx.defender)
	if defPokemon.Ability.Name == "suction-cups" {
		return []StateEvent{
			SimpleAbilityActivationEvent(&ctx.gameState, ctx.defender),
			NewFmtMessageEvent("%s cannot be forced out!", defPokemon.Name()),
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

	rng := ctx.gameState.CreateRng()

	choiceIndex := rng.IntN(len(alivePokemon))

	return []StateEvent{
		SwitchEvent{PlayerIndex: ctx.defender, SwitchIndex: alivePokemon[choiceIndex].Index},
	}
}

func (a AttackAction) GetCtx() ActionCtx {
	return a.Ctx
}
