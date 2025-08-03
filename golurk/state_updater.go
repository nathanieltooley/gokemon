package golurk

import (
	"fmt"
	"reflect"
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

	events = append(events, commonSwitching(*gameState, switches)...)

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

	events = append(events, commonOtherActionHandling(*gameState, otherActions)...)

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

	events = append(events, commonEndOfTurn(gameState)...)

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
