package golurk

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"

	"github.com/samber/lo"
)

const (
	RESULT_RESOLVED = iota + 1
	RESULT_GAMEOVER
	RESULT_FORCESWITCH
)

// TurnResult represents the result of a turn or part of a turn (in the case of a force switch)
// Unlike events, TurnResult is a single struct with a tag, Kind, that determines the result.
//
// If Go had better support for tagged unions, enums or better Un/Marshaling of interfaces, they would be the same.
// However, there shouldn't be that many different kinds results and I most likely have already enumerated all possibilites.
// If this is not the case I will eat my shoe.
type TurnResult struct {
	// dude imagine if go had tagged unions that would be crazy
	// i'd take enums to tbh

	Kind   int
	Events []StateEvent
	// Only used for RESULT_GAMEOVER and RESULT_FORCESWITCH
	ForThisPlayer bool
}

func ProcessTurn(gameState *GameState, actions []Action) TurnResult {
	host := &gameState.HostPlayer
	client := &gameState.ClientPlayer

	switches := make([]SwitchAction, 0)
	otherActions := make([]Action, 0)

	events := make([]StateEvent, 0)

	backFromForceSwitch := host.ActiveKOed || client.ActiveKOed

	// Sort different actions
	for _, a := range actions {
		switch a := a.(type) {
		case SwitchAction:
			switches = append(switches, a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	hostPokemon := host.GetActivePokemon()
	clientPokemon := client.GetActivePokemon()

	if !backFromForceSwitch {
		internalLogger.WithName("state_updater").Info(fmt.Sprintf("\n\n======== TURN %d =========", gameState.Turn))
		// Reset turn flags
		// eventually this will have to change for double battles
		// NOTE: i want to keep updates outside of events like this rare. i will make an exception here there is no visual
		// for when a pokemon can't attack and it saves us from adding an attack action that would have to been skipped while iterating through them
		hostPokemon.CanAttackThisTurn = true
		hostPokemon.SwitchedInThisTurn = false

		clientPokemon.CanAttackThisTurn = true
		clientPokemon.SwitchedInThisTurn = false
	}

	for _, action := range actions {
		internalLogger.V(1).Info("Player Action", "player_id", action.GetCtx().PlayerID, "action_name", reflect.TypeOf(action).Name())
	}

	events = append(events, switchEvents(*gameState, switches)...)

	// Properly end turn after force switches are dealt with
	if backFromForceSwitch {
		internalLogger.V(1).Info("coming back from force switch")
		// TODO: Force updater to switch out a pokemon if current, and also dead, pokemon is not switched out
		host.ActiveKOed = false
		client.ActiveKOed = false

		gameState.Turn++

		return TurnResult{
			Kind:   RESULT_RESOLVED,
			Events: events,
		}
	}

	if hostPokemon.Ability.Name == "truant" && hostPokemon.TruantShouldActivate {
		events = append(events, SimpleAbilityActivationEvent(gameState, HOST))
		// NOTE: see previous note though im thinking it might end up being fine?
		hostPokemon.CanAttackThisTurn = false
	}

	if clientPokemon.Ability.Name == "truant" && clientPokemon.TruantShouldActivate {
		events = append(events, SimpleAbilityActivationEvent(gameState, PEER))
		clientPokemon.CanAttackThisTurn = false
	}

	events = append(events, actionEvents(*gameState, otherActions)...)

	// we don't want to modify the original state just yet but we need play through what events have already happened
	clonedState := gameState.Clone()
	ApplyEventsToState(&clonedState, TurnResult{
		Kind:   RESULT_RESOLVED,
		Events: events,
	})

	gameOverValue := clonedState.GameOver()
	switch gameOverValue {
	case HOST:
		return TurnResult{
			Kind:          RESULT_GAMEOVER,
			ForThisPlayer: true,
			Events:        events,
		}
	case PEER:
		return TurnResult{
			Kind:          RESULT_GAMEOVER,
			ForThisPlayer: false,
			Events:        events,
		}
	}

	if !clonedState.HostPlayer.GetActivePokemon().Alive() {
		host.ActiveKOed = true
		internalLogger.V(1).Info("host's pokemon has been killed. returning force switch")
		return TurnResult{
			Kind:          RESULT_FORCESWITCH,
			ForThisPlayer: true,
			Events:        events,
		}
	}

	if !clonedState.ClientPlayer.GetActivePokemon().Alive() {
		client.ActiveKOed = true
		internalLogger.V(1).Info("client's pokemon has been killed. returning force switch")
		return TurnResult{
			Kind:          RESULT_FORCESWITCH,
			ForThisPlayer: false,
			Events:        events,
		}
	}

	events = append(events, endOfTurnEvents(gameState)...)

	gameState.Turn++

	return TurnResult{
		Kind:   RESULT_RESOLVED,
		Events: events,
	}
}

func ApplyEventsToState(gameState *GameState, result TurnResult) {
	eventIter := NewEventIter()
	eventIter.AddEvents(result.Events)

	for {
		_, next := eventIter.Next(gameState)
		if !next {
			break
		}
	}
}

func switchEvents(gameState GameState, switches []SwitchAction) []StateEvent {
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

func actionEvents(gameState GameState, actions []Action) []StateEvent {
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
		case AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case SkipAction, *SkipAction:
			aPriority = -100
		default:
			internalLogger.Error(fmt.Errorf("unaccounted for action while trying to sort action"), "")
			return 0
		}

		activePokemon = gameState.GetPlayer(b.GetCtx().PlayerID).GetActivePokemon()
		bSpeed = activePokemon.Speed(gameState.Weather)

		switch b := b.(type) {
		case AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case SkipAction:
			bPriority = -100
		default:
			internalLogger.Error(fmt.Errorf("unaccounted for action while trying to sort action"), "")
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

				internalLogger.Info("attack was skipped because of para", "pokemon_name", pokemon.Name())

				events = append(events, paraEvent)
				return
			}

			// Skip attack with sleep
			if pokemon.Status == STATUS_SLEEP {
				sleepEv := SleepEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of sleep", "pokemon_name", pokemon.Name())

				events = append(events, sleepEv)
				return
			}

			// Skip attack with frozen
			if pokemon.Status == STATUS_FROZEN {
				frzEv := FrozenEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of freeze", "pokemon_name", pokemon.Name())

				events = append(events, frzEv)
				return
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				confusionEv := ConfusionEvent{
					PlayerIndex:         a.GetCtx().PlayerID,
					FollowUpAttackEvent: a.UpdateState(gameState)[0],
				}

				internalLogger.Info("attack was skipped because of confusion", "pokemon_name", pokemon.Name())

				events = append(events, confusionEv)
				return
			}

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				events = append(events, a.UpdateState(gameState)...)
			} else if !pokemon.Alive() {
				internalLogger.Info("attack was skipped because of dead", "pokemon_name", pokemon.Name())
			} else if !pokemon.CanAttackThisTurn {
				internalLogger.Info("attack was skipped because it was marked as unable to attack for the turn", "pokemon_name", pokemon.Name())
			}
		default:
			events = append(events, a.UpdateState(gameState)...)
		}
	})

	return events
}

func endOfTurnEvents(gameState *GameState) []StateEvent {
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

	events = append(events, FinalUpdatesEvent{})

	events = append(events, EndOfTurnAbilityCheck{PlayerID: HOST})
	events = append(events, EndOfTurnAbilityCheck{PlayerID: PEER})

	return events
}
