package golurk

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/networking"
	"github.com/rs/zerolog/log"
)

func ProcessTurn(gameState *stateCore.GameState, actions []stateCore.Action) tea.Msg {
	host := &gameState.HostPlayer
	client := &gameState.ClientPlayer

	switches := make([]stateCore.SwitchAction, 0)
	otherActions := make([]stateCore.Action, 0)

	events := make([]stateCore.StateEvent, 0)

	backFromForceSwitch := host.ActiveKOed || client.ActiveKOed

	// Sort different actions
	for _, a := range actions {
		switch a := a.(type) {
		case stateCore.SwitchAction:
			switches = append(switches, a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	hostPokemon := host.GetActivePokemon()
	clientPokemon := client.GetActivePokemon()

	if !backFromForceSwitch {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
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
		log.Info().Msgf("Player Action: %s", reflect.TypeOf(action).Name())
	}

	events = append(events, commonSwitching(*gameState, switches)...)

	// Properly end turn after force switches are dealt with
	if backFromForceSwitch {
		log.Info().Msg("coming back from force switch")
		// TODO: Force updater to switch out a pokemon if current, and also dead, pokemon is not switched out
		host.ActiveKOed = false
		client.ActiveKOed = false

		gameState.Turn++

		log.Info().Msgf("Events: %d", len(events))

		return networking.TurnResolvedMessage{
			Events: networking.EventSlice{Events: events},
		}
	}

	if hostPokemon.Ability.Name == "truant" && hostPokemon.TruantShouldActivate {
		events = append(events, stateCore.SimpleAbilityActivationEvent(gameState, stateCore.HOST))
		// NOTE: see previous note though im thinking it might end up being fine?
		hostPokemon.CanAttackThisTurn = false
	}

	if clientPokemon.Ability.Name == "truant" && clientPokemon.TruantShouldActivate {
		events = append(events, stateCore.SimpleAbilityActivationEvent(gameState, stateCore.PEER))
		clientPokemon.CanAttackThisTurn = false
	}

	events = append(events, commonOtherActionHandling(*gameState, otherActions)...)

	// we don't want to modify the original state just yet but we need play through what events have already happened
	clonedState := gameState.Clone()
	ApplyEventsToState(&clonedState, networking.TurnResolvedMessage{
		Events: networking.EventSlice{Events: events},
	})

	gameOverValue := clonedState.GameOver()
	switch gameOverValue {
	case stateCore.HOST:
		return networking.GameOverMessage{
			YouLost: true,
			Events:  networking.EventSlice{Events: events},
		}
	case stateCore.PEER:
		return networking.GameOverMessage{
			YouLost: false,
			Events:  networking.EventSlice{Events: events},
		}
	}

	if !clonedState.HostPlayer.GetActivePokemon().Alive() {
		host.ActiveKOed = true
		log.Debug().Msg("host's pokemon has been killed. returning force switch")
		return networking.ForceSwitchMessage{
			ForThisPlayer: true,
			Events:        networking.EventSlice{Events: events},
		}
	}

	if !clonedState.ClientPlayer.GetActivePokemon().Alive() {
		client.ActiveKOed = true
		log.Debug().Msg("client's pokemon has been killed. returning force switch")
		return networking.ForceSwitchMessage{
			ForThisPlayer: false,
			Events:        networking.EventSlice{Events: events},
		}
	}

	events = append(events, commonEndOfTurn(gameState)...)

	gameState.Turn++

	return networking.TurnResolvedMessage{
		Events: networking.EventSlice{Events: events},
	}
}

func ApplyEventsToState(gameState *stateCore.GameState, msg tea.Msg) {
	turnEndMsg := msg.(networking.TurnResolvedMessage)
	eventIter := stateCore.NewEventIter()
	eventIter.AddEvents(turnEndMsg.Events.Events)

	for {
		_, next := eventIter.Next(gameState)
		if !next {
			break
		}
	}
}
