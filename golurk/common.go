package golurk

import (
	"cmp"
	"slices"

	"github.com/samber/lo"
)

func commonSwitching(gameState GameState, switches []SwitchAction) []StateEvent {
	events := make([]StateEvent, 0)

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b SwitchAction) int {
		return cmp.Compare(a.Poke.Speed(gameState.Weather), b.Poke.Speed(gameState.Weather))
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a SwitchAction, i int) {
		events = append(events, a.UpdateState(gameState)...)
	})

	return events
}

func commonOtherActionHandling(gameState GameState, actions []Action) []StateEvent {
	events := make([]StateEvent, 0)

	// Sort Other Actions
	// TODO: Fix this so that instead of sorting ahead of time, whenever an action is processed, it grabs the "fastest" action next.
	// This way, previous actions that change speed can affect the order of following actions. This will mainly be important for double battles.
	slices.SortFunc(actions, func(a, b Action) int {
		var aSpeed int
		var bSpeed int
		var aPriority int
		var bPriority int

		activePokemon := gameState.GetPlayer(a.GetCtx().PlayerID).GetActivePokemon()
		aSpeed = activePokemon.Speed(gameState.Weather)

		switch a := a.(type) {
		case *AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case *SkipAction:
			aPriority = -100
		default:
			return 0
		}

		activePokemon = gameState.GetPlayer(b.GetCtx().PlayerID).GetActivePokemon()
		bSpeed = activePokemon.Speed(gameState.Weather)

		switch b := b.(type) {
		case *AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case *SkipAction:
			bPriority = -100
		default:
			return 0
		}

		internalLogger.V(2).Info("sort debug",
			"aPlayer", a.GetCtx().PlayerID,
			"bPlayer", b.GetCtx().PlayerID,
			"aSpeed", aSpeed,
			"bSpeed", bSpeed,
			"aPriority", aPriority,
			"bPriority", bPriority,
			"comp", cmp.Compare(aSpeed, bSpeed),
			"compPriority", cmp.Compare(aPriority, bPriority),
		)

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
	lo.ForEach(actions, func(a Action, i int) {
		switch a.(type) {
		case AttackAction, *AttackAction, SkipAction, *SkipAction:
			player := gameState.GetPlayer(a.GetCtx().PlayerID)

			internalLogger.V(2).Info("attack state update",
				"attackIndex", i,
				"attackerSpeed", player.GetActivePokemon().Speed(gameState.Weather),
				"attackerRawSpeed", player.GetActivePokemon().RawSpeed.CalcValue(),
				"attackerConfCount", player.GetActivePokemon().ConfusionCount,
			)

			pokemon := player.GetActivePokemon()
			if pokemon.CanAttackThisTurn {
				pokemon.CanAttackThisTurn = !pokemon.SwitchedInThisTurn
			}

			if !pokemon.Alive() {
				return
			}

			// Skip attack with para
			if pokemon.Status == STATUS_PARA {
				paraEvent := ParaEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of para", "pokemon_name", pokemon.Nickname)

				events = append(events, paraEvent)
				return
			}

			// Skip attack with sleep
			if pokemon.Status == STATUS_SLEEP {
				sleepEv := SleepEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of sleep", "pokemon_name", pokemon.Nickname)

				events = append(events, sleepEv)
				return
			}

			// Skip attack with frozen
			if pokemon.Status == STATUS_FROZEN {
				frzEv := FrozenEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of freeze", "pokemon_name", pokemon.Nickname)

				events = append(events, frzEv)
				return
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				confusionEv := ConfusionEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of confusion", "pokemon_name", pokemon.Nickname)

				events = append(events, confusionEv)
				return
			}

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				events = append(events, a.UpdateState(gameState)...)
			} else if !pokemon.Alive() {
				internalLogger.Info("attack was skipped because of dead", "pokemon_name", pokemon.Nickname)
			} else if !pokemon.CanAttackThisTurn {
				internalLogger.Info("attack was skipped because it was marked as unable to attack for the turn", "pokemon_name", pokemon.Nickname)
			}
		default:
			events = append(events, a.UpdateState(gameState)...)
		}
	})

	return events
}

func commonEndOfTurn(gameState *GameState) []StateEvent {
	events := make([]StateEvent, 0)

	applyBurn := func(index int) {
		events = append(events, BurnEvent{PlayerIndex: index})
	}

	applyPoison := func(index int) {
		events = append(events, PoisonEvent{PlayerIndex: index})
	}

	applyToxic := func(index int) {
		events = append(events, ToxicEvent{PlayerIndex: index})
	}

	localPokemon := gameState.HostPlayer.GetActivePokemon()
	switch localPokemon.Status {
	case STATUS_BURN:
		applyBurn(HOST)
	case STATUS_POISON:
		applyPoison(HOST)
	case STATUS_TOXIC:
		applyToxic(HOST)
	}

	opPokemon := gameState.ClientPlayer.GetActivePokemon()
	switch opPokemon.Status {
	case STATUS_BURN:
		applyBurn(PEER)
	case STATUS_POISON:
		applyPoison(PEER)
	case STATUS_TOXIC:
		applyToxic(PEER)
	}

	if gameState.Weather == WEATHER_SANDSTORM {
		events = append(events, SandstormDamageEvent{PlayerIndex: HOST})
		events = append(events, SandstormDamageEvent{PlayerIndex: PEER})
	}

	events = append(events, EndOfTurnAbilityCheck{PlayerID: HOST})
	events = append(events, EndOfTurnAbilityCheck{PlayerID: PEER})

	return events
}

func InvertPlayerIndex(initial int) int {
	if initial == HOST {
		return PEER
	} else {
		return HOST
	}
}
