package state

import (
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

	if !backFromForceSwitch {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
		events = append(events, stateCore.TurnStartEvent{})
	}

	events = append(events, commonSwitching(*gameState, switches)...)

	// Properly end turn after force switches are dealt with
	if backFromForceSwitch {
		host.ActiveKOed = false
		client.ActiveKOed = false

		gameState.Turn++

		log.Info().Msgf("Events: %d", len(events))

		return networking.TurnResolvedMessage{
			Events: networking.EventSlice{Events: events},
		}
	}

	hostPokemon := host.GetActivePokemon()
	if hostPokemon.Ability.Name == "truant" && hostPokemon.TruantShouldActivate {
		events = append(events, stateCore.SimpleAbilityActivationEvent(gameState, stateCore.HOST))
		// NOTE: i want to keep updates outside of events like this rare. i will make an exception here there is no visual
		// for when a pokemon can't attack and it saves us from adding an action that would have to been skipped while iterating through them
		hostPokemon.CanAttackThisTurn = false
	}

	clientPokemon := client.GetActivePokemon()
	if clientPokemon.Ability.Name == "truant" && clientPokemon.TruantShouldActivate {
		events = append(events, stateCore.SimpleAbilityActivationEvent(gameState, stateCore.PEER))
		clientPokemon.CanAttackThisTurn = false
	}

	events = append(events, commonOtherActionHandling(*gameState, otherActions)...)

	gameOverValue := gameState.GameOver()
	switch gameOverValue {
	case PLAYER:
		return networking.GameOverMessage{
			YouLost: true,
		}
	case PEER:
		return networking.GameOverMessage{
			YouLost: false,
		}
	}

	// Seems weird but should make sense if or when multiplayer is added
	// also these checks will have to change if double battles are added
	if !gameState.HostPlayer.GetActivePokemon().Alive() {
		host.ActiveKOed = true
		return networking.ForceSwitchMessage{
			ForThisPlayer: true,
			Events:        networking.EventSlice{Events: events},
		}
	}

	if !gameState.ClientPlayer.GetActivePokemon().Alive() {
		client.ActiveKOed = true
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
