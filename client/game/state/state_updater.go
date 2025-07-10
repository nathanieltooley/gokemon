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

	// Sort different actions
	for _, a := range actions {
		switch a := a.(type) {
		case *stateCore.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	events = append(events, commonSwitching(*gameState, switches)...)

	// Properly end turn after force switches are dealt with
	if host.ActiveKOed || client.ActiveKOed {
		// wish i didn't have to deal with cleaning up state here
		host.ActiveKOed = false
		client.ActiveKOed = false

		gameState.Turn++

		log.Info().Msgf("Events: %d", len(events))

		return networking.TurnResolvedMessage{
			Events: events,
		}
	} else {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
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
			Events:        events,
		}
	}

	if !gameState.ClientPlayer.GetActivePokemon().Alive() {
		client.ActiveKOed = true
		return networking.ForceSwitchMessage{
			ForThisPlayer: false,
			Events:        events,
		}
	}

	events = append(events, commonEndOfTurn(gameState)...)

	gameState.Turn++

	return networking.TurnResolvedMessage{
		Events: events,
	}
}
