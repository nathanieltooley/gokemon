package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/networking"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// Takes a list of state snapshots and applies the final state to the main copy of state,
// syncing all intermediate changes with the main state and returning the snapshots
func syncState(mainState *stateCore.GameState, newStates []stateCore.StateSnapshot) []stateCore.StateSnapshot {
	finalState := *mainState

	// get first (from end) non-empty state
	// we don't need to apply every state update, just the last one
	// the UI is what's interested in the intermediate states
	for i := len(newStates) - 1; i >= 0; i-- {
		s := newStates[i]
		if !s.Empty && !s.MessagesOnly {
			finalState = s.State
			break
		}
	}

	// Clone here because of slices
	*mainState = finalState.Clone()

	return newStates
}

// Removes empty state snapshots and combines message only state updates with the previous full state update
func cleanStateSnapshots(snaps []stateCore.StateSnapshot) []stateCore.StateSnapshot {
	// Ignore empty state updates
	snaps = lo.Filter(snaps, func(s stateCore.StateSnapshot, _ int) bool {
		return !s.Empty
	})

	if len(snaps) <= 1 {
		return snaps
	}

	// if a snapshot only contains messages, append them to the last real snapshot
	previousSnap := &snaps[0]
	for i := 1; i < len(snaps); i++ {
		currentSnap := snaps[i]
		if currentSnap.MessagesOnly {
			previousSnap.Messages = append(previousSnap.Messages, currentSnap.Messages...)
		} else {
			previousSnap = &snaps[i]
		}
	}

	// Get rid of message only snapshots
	return lo.Filter(snaps, func(s stateCore.StateSnapshot, _ int) bool {
		return !s.MessagesOnly
	})
}

func ProcessTurn(gameState *stateCore.GameState, actions []stateCore.Action) tea.Msg {
	host := &gameState.HostPlayer
	client := &gameState.ClientPlayer

	switches := make([]stateCore.SwitchAction, 0)
	otherActions := make([]stateCore.Action, 0)

	states := make([]stateCore.StateSnapshot, 0)

	// Sort different actions
	for _, a := range actions {
		switch a := a.(type) {
		case *stateCore.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	states = append(states, syncState(gameState, commonSwitching(gameState, switches))...)

	// Properly end turn after force switches are dealt with
	if host.ActiveKOed || client.ActiveKOed {
		// wish i didn't have to deal with cleaning up state here
		host.ActiveKOed = false
		client.ActiveKOed = false

		gameState.Turn++

		messages := lo.FlatMap(states, func(item stateCore.StateSnapshot, i int) []string {
			return item.Messages
		})

		log.Info().Msgf("States: %d", len(states))
		log.Info().Strs("Queued Messages", messages).Msg("")

		if len(states) != 0 {
			// HACK: same one as above
			cleanedStates := cleanStateSnapshots(states)
			finalState := cleanedStates[len(cleanedStates)-1]
			finalState.State.Turn = gameState.Turn
			cleanedStates[len(cleanedStates)-1] = finalState
		}

		gameState.MessageHistory = append(gameState.MessageHistory, messages...)

		return networking.TurnResolvedMessage{
			StateUpdates: cleanStateSnapshots(states),
		}
	} else {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
	}

	states = append(states, syncState(gameState, commonOtherActionHandling(gameState, otherActions))...)

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
			StateUpdates:  cleanStateSnapshots(states),
		}
	}

	if !gameState.ClientPlayer.GetActivePokemon().Alive() {
		client.ActiveKOed = true
		return networking.ForceSwitchMessage{
			ForThisPlayer: false,
			StateUpdates:  cleanStateSnapshots(states),
		}
	}

	states = append(states, syncState(gameState, commonEndOfTurn(gameState))...)

	gameState.Turn++

	if len(states) != 0 {
		// HACK: same one as above
		cleanedStates := cleanStateSnapshots(states)
		finalState := cleanedStates[len(cleanedStates)-1]
		finalState.State.Turn = gameState.Turn
		cleanedStates[len(cleanedStates)-1] = finalState
	}

	return networking.TurnResolvedMessage{
		StateUpdates: cleanStateSnapshots(states),
	}
}

// Activates certain end of turn abilities
func endOfTurnAbilities(gameState *stateCore.GameState, player int) []stateCore.StateSnapshot {
	playerPokemon := gameState.GetPlayer(player).GetActivePokemon()

	states := make([]stateCore.StateSnapshot, 0)

	abilityText := fmt.Sprintf("%s activated their ability: %s", playerPokemon.Nickname, playerPokemon.Ability.Name)

	switch playerPokemon.Ability.Name {
	// TEST: no gen 1 pkm have this ability
	case "speed-boost":
		if !playerPokemon.SwitchedInThisTurn {
			states = append(states, stateCore.NewMessageOnlySnapshot(abilityText))
			states = append(states, stateCore.StatChangeHandler(gameState, playerPokemon, core.StatChange{Change: 1, StatName: "speed"}, 100))
		}
	}

	return states
}
