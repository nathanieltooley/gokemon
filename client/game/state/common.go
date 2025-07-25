package state

import (
	"cmp"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func commonSwitching(gameState stateCore.GameState, switches []stateCore.SwitchAction) []stateCore.StateEvent {
	events := make([]stateCore.StateEvent, 0)

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b stateCore.SwitchAction) int {
		return cmp.Compare(a.Poke.Speed(gameState.Weather), b.Poke.Speed(gameState.Weather))
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a stateCore.SwitchAction, i int) {
		events = append(events, a.UpdateState(gameState)...)
	})

	return events
}

func commonOtherActionHandling(gameState stateCore.GameState, actions []stateCore.Action) []stateCore.StateEvent {
	events := make([]stateCore.StateEvent, 0)

	// Sort Other Actions
	// TODO: Fix this so that instead of sorting ahead of time, whenever an action is processed, it grabs the "fastest" action next.
	// This way, previous actions that change speed can affect the order of following actions. This will mainly be important for double battles.
	slices.SortFunc(actions, func(a, b stateCore.Action) int {
		var aSpeed int
		var bSpeed int
		var aPriority int
		var bPriority int

		activePokemon := gameState.GetPlayer(a.GetCtx().PlayerID).GetActivePokemon()
		aSpeed = activePokemon.Speed(gameState.Weather)

		switch a := a.(type) {
		case *stateCore.AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case *stateCore.SkipAction:
			aPriority = -100
		default:
			return 0
		}

		activePokemon = gameState.GetPlayer(b.GetCtx().PlayerID).GetActivePokemon()
		bSpeed = activePokemon.Speed(gameState.Weather)

		switch b := b.(type) {
		case *stateCore.AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case *stateCore.SkipAction:
			bPriority = -100
		default:
			return 0
		}

		log.Debug().
			Int("aPlayer", a.GetCtx().PlayerID).
			Int("bPlayer", b.GetCtx().PlayerID).
			Int("aSpeed", aSpeed).
			Int("bSpeed", bSpeed).
			Int("aPriority", aPriority).
			Int("bPriority", bPriority).
			Int("comp", cmp.Compare(aSpeed, bSpeed)).
			Int("compPriority", cmp.Compare(aPriority, bPriority)).
			Msg("sort debug")

		priorComp := cmp.Compare(aPriority, bPriority)
		speedComp := cmp.Compare(aSpeed, bSpeed)

		if priorComp == 0 {
			return speedComp
		} else {
			return priorComp
		}
	})

	// Reverse for desc order
	slices.Reverse(actions)

	// Process otherActions next
	lo.ForEach(actions, func(a stateCore.Action, i int) {
		switch a.(type) {
		case stateCore.AttackAction, *stateCore.AttackAction, stateCore.SkipAction, *stateCore.SkipAction:
			player := gameState.GetPlayer(a.GetCtx().PlayerID)

			log.Info().Int("attackIndex", i).
				Int("attackerSpeed", player.GetActivePokemon().Speed(gameState.Weather)).
				Int("attackerRawSpeed", player.GetActivePokemon().RawSpeed.CalcValue()).
				Int("attackerConfCount", player.GetActivePokemon().ConfusionCount).
				Msg("Attack state update")

			pokemon := player.GetActivePokemon()
			if pokemon.CanAttackThisTurn {
				pokemon.CanAttackThisTurn = !pokemon.SwitchedInThisTurn
			}

			if !pokemon.Alive() {
				return
			}

			// Skip attack with para
			if pokemon.Status == core.STATUS_PARA {
				paraEvent := stateCore.ParaEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				log.Info().Msgf("%s attack was skipped because of para", pokemon.Nickname)

				events = append(events, paraEvent)
				return
			}

			// Skip attack with sleep
			if pokemon.Status == core.STATUS_SLEEP {
				sleepEv := stateCore.SleepEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				log.Info().Msgf("%s attack was skipped because of sleep", pokemon.Nickname)

				events = append(events, sleepEv)
				return
			}

			// Skip attack with frozen
			if pokemon.Status == core.STATUS_FROZEN {
				frzEv := stateCore.FrozenEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				log.Info().Msgf("%s attack was skipped because of freeze", pokemon.Nickname)

				events = append(events, frzEv)
				return
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				confusionEv := stateCore.ConfusionEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				log.Info().Msgf("%s attack was skipped because of confusion", pokemon.Nickname)

				events = append(events, confusionEv)
				return
			}

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				events = append(events, a.UpdateState(gameState)...)
			} else if !pokemon.Alive() {
				log.Info().Msgf("%s attack was skipped because of dead", pokemon.Nickname)
			} else if !pokemon.CanAttackThisTurn {
				log.Info().Msgf("%s attack was skipped because it was marked as unable to attack for the turn", pokemon.Nickname)
			}
		default:
			events = append(events, a.UpdateState(gameState)...)
		}
	})

	return events
}

func commonEndOfTurn(gameState *stateCore.GameState) []stateCore.StateEvent {
	events := make([]stateCore.StateEvent, 0)

	applyBurn := func(index int) {
		events = append(events, stateCore.BurnEvent{PlayerIndex: index})
	}

	applyPoison := func(index int) {
		events = append(events, stateCore.PoisonEvent{PlayerIndex: index})
	}

	applyToxic := func(index int) {
		events = append(events, stateCore.ToxicEvent{PlayerIndex: index})
	}

	localPokemon := gameState.HostPlayer.GetActivePokemon()
	switch localPokemon.Status {
	case core.STATUS_BURN:
		applyBurn(stateCore.HOST)
	case core.STATUS_POISON:
		applyPoison(stateCore.HOST)
	case core.STATUS_TOXIC:
		applyToxic(stateCore.HOST)
	}

	opPokemon := gameState.ClientPlayer.GetActivePokemon()
	switch opPokemon.Status {
	case core.STATUS_BURN:
		applyBurn(stateCore.PEER)
	case core.STATUS_POISON:
		applyPoison(stateCore.PEER)
	case core.STATUS_TOXIC:
		applyToxic(stateCore.PEER)
	}

	if gameState.Weather == core.WEATHER_SANDSTORM {
		events = append(events, stateCore.SandstormDamageEvent{PlayerIndex: stateCore.HOST})
		events = append(events, stateCore.SandstormDamageEvent{PlayerIndex: stateCore.PEER})
	}

	events = append(events, stateCore.EndOfTurnAbilityCheck{PlayerID: stateCore.HOST})
	events = append(events, stateCore.EndOfTurnAbilityCheck{PlayerID: stateCore.PEER})

	return events
}
