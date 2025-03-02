package stateupdater

import (
	"fmt"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/networking"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// Abstraction of state changes (pokemon taking dmg, statuses changing, weather updates, etc).
// The reason for this abstraction is to hide away any network dependent stuff from the main UI code.
// Singleplayer games return artifically delayed cmds after updates while networked games will have actual latency
type StateUpdater func(*state.GameState, []state.Action) tea.Cmd

// Takes a list of state snapshots and applies the final state to the main copy of state,
// syncing all intermediate changes with the main state and returning the snapshots
func syncState(mainState *state.GameState, newStates []state.StateSnapshot) []state.StateSnapshot {
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
func cleanStateSnapshots(snaps []state.StateSnapshot) []state.StateSnapshot {
	// Ignore empty state updates
	snaps = lo.Filter(snaps, func(s state.StateSnapshot, _ int) bool {
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
	return lo.Filter(snaps, func(s state.StateSnapshot, _ int) bool {
		return !s.MessagesOnly
	})
}

// Determines the best AI Action. Failsafes to skip actions
func bestAiAction(gameState *state.GameState) state.Action {
	if gameState.OpposingPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.LocalPlayer.GetActivePokemon()
		aiPokemon := gameState.OpposingPlayer.GetActivePokemon()

		bestMoveIndex := 0
		var bestMove game.Move
		var bestMoveDamage uint = 0

		for i, move := range aiPokemon.Moves {
			if move.IsNil() {
				continue
			}

			// assume no crits
			moveDamage := game.Damage(*aiPokemon, *playerPokemon, move, false, gameState.Weather)
			if moveDamage > bestMoveDamage {
				bestMoveIndex = i
				bestMove = move
				bestMoveDamage = moveDamage
			}
		}

		if bestMove.IsNil() {
			return &state.SkipAction{}
		} else {
			return state.NewAttackAction(state.AI, bestMoveIndex)
		}

	} else {
		// Switch on death
		for i, pokemon := range gameState.OpposingPlayer.Team {
			if pokemon.Alive() {
				return state.NewSwitchAction(gameState, state.AI, i)
			}
		}
	}

	return &state.SkipAction{}
}

// The updater for singleplayer games.
// Introduces artifical delay so theres some space in between human actions
func LocalUpdater(gameState *state.GameState, actions []state.Action) tea.Cmd {
	artificalDelay := time.Second * 2

	host := &gameState.LocalPlayer
	ai := &gameState.OpposingPlayer

	// Do not have the AI add a new action to the list if the player is switching and the AI isnt
	if !host.ActiveKOed || ai.ActiveKOed {
		actions = append(actions, bestAiAction(gameState))
	}

	switches := make([]state.SwitchAction, 0)
	otherActions := make([]state.Action, 0)

	states := make([]state.StateSnapshot, 0)

	// Sort different actions
	for _, a := range actions {
		switch a := a.(type) {
		case *state.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	states = append(states, commonSwitching(gameState, switches)...)

	// Properly end turn after force switches are dealt with
	if host.ActiveKOed || ai.ActiveKOed {
		// wish i didn't have to deal with cleaning up state here
		host.ActiveKOed = false
		ai.ActiveKOed = false

		gameState.Turn++

		return func() tea.Msg {
			time.Sleep(time.Second * 1)

			messages := lo.FlatMap(states, func(item state.StateSnapshot, i int) []string {
				return item.Messages
			})

			log.Info().Msgf("States: %d", len(states))
			log.Info().Strs("Queued Messages", messages).Msg("")

			gameState.MessageHistory = append(gameState.MessageHistory, messages...)

			return TurnResolvedMessage{
				StateUpdates: cleanStateSnapshots(states),
			}
		}
	} else {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
	}

	states = append(states, commonOtherActionHandling(gameState, otherActions)...)

	gameOverValue := gameState.GameOver()
	if gameOverValue == state.PLAYER {
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return GameOverMessage{
				ForThisPlayer: true,
			}
		}
	} else if gameOverValue == state.AI {
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return GameOverMessage{
				ForThisPlayer: false,
			}
		}
	}

	// Seems weird but should make sense if or when multiplayer is added
	// also these checks will have to change if double battles are added
	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		host.ActiveKOed = true
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return ForceSwitchMessage{
				ForThisPlayer: true,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
		ai.ActiveKOed = true
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	states = append(states, commonEndOfTurn(gameState)...)

	return func() tea.Msg {
		// Artifical Delay
		time.Sleep(artificalDelay)

		gameState.Turn++

		return TurnResolvedMessage{
			StateUpdates: cleanStateSnapshots(states),
		}
	}
}

func NetHostUpdater(gameState *state.GameState, actions []state.Action, conn net.Conn) tea.Cmd {
	host := &gameState.LocalPlayer
	op := &gameState.OpposingPlayer

	// We need to get an action from the opposing player
	if !host.ActiveKOed || op.ActiveKOed {
		// TODO: Have this return a tea.Cmd. gameState will have to keep track of when the players have submitted actions

		// GET ACTION
		opAction, err := networking.AcceptAction(conn)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get opposing action")
		}

		log.Info().Msgf("Host got action: %+v", opAction)

		actions = append(actions, opAction)
	}

	switches := make([]state.SwitchAction, 0)
	otherActions := make([]state.Action, 0)

	states := make([]state.StateSnapshot, 0)

	for _, a := range actions {
		switch a := a.(type) {
		case *state.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	states = append(states, commonSwitching(gameState, switches)...)

	if host.ActiveKOed || op.ActiveKOed {
		// wish i didn't have to deal with cleaning up state here
		host.ActiveKOed = false
		op.ActiveKOed = false

		gameState.Turn++

		return func() tea.Msg {
			time.Sleep(time.Second * 1)

			messages := lo.FlatMap(states, func(item state.StateSnapshot, i int) []string {
				return item.Messages
			})

			log.Info().Msgf("States: %d", len(states))
			log.Info().Strs("Queued Messages", messages).Msg("")

			gameState.MessageHistory = append(gameState.MessageHistory, messages...)

			networking.SendData(conn, TurnResolvedMessage{
				StateUpdates: cleanStateSnapshots(states),
			})

			return TurnResolvedMessage{
				StateUpdates: cleanStateSnapshots(states),
			}
		}
	} else {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
	}

	states = append(states, commonOtherActionHandling(gameState, otherActions)...)

	gameOverValue := gameState.GameOver()
	if gameOverValue == state.PLAYER {
		return func() tea.Msg {
			networking.SendData(conn, GameOverMessage{
				ForThisPlayer: false,
			})

			return GameOverMessage{
				ForThisPlayer: true,
			}
		}
	} else if gameOverValue == state.PEER {
		return func() tea.Msg {
			networking.SendData(conn, GameOverMessage{
				ForThisPlayer: true,
			})

			return GameOverMessage{
				ForThisPlayer: false,
			}
		}
	}

	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		host.ActiveKOed = true
		return func() tea.Msg {
			networking.SendData(conn, ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  cleanStateSnapshots(states),
			})

			return ForceSwitchMessage{
				ForThisPlayer: true,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
		op.ActiveKOed = true
		return func() tea.Msg {

			networking.SendData(conn, ForceSwitchMessage{
				ForThisPlayer: true,
				StateUpdates:  cleanStateSnapshots(states),
			})

			return ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	states = append(states, commonEndOfTurn(gameState)...)

	return func() tea.Msg {
		networking.SendData(conn, TurnResolvedMessage{
			StateUpdates: cleanStateSnapshots(states),
		})

		gameState.Turn++

		return TurnResolvedMessage{
			StateUpdates: cleanStateSnapshots(states),
		}
	}
}

func NetClientUpdater(gameState *state.GameState, actions []state.Action, conn net.Conn) tea.Cmd {
	// the client is only going to have action,
	// send that to the host, and then get all of the state updates
	if len(actions) != 1 {
		log.Fatal().Msg("client tried to update with the incorrect amount of actions")
	}

	if err := networking.SendAction(conn, actions[0]); err != nil {
		log.Fatal().Err(err).Msg("client failed to send action")
	}

	_, err := networking.AcceptData[[]state.StateSnapshot](conn)

	if err != nil {
		log.Fatal().Err(err).Msg("client failed to receive state updates")
	}

	return nil
}

// Activates certain end of turn abilities
func endOfTurnAbilities(gameState *state.GameState, player int) []state.StateSnapshot {
	playerPokemon := gameState.GetPlayer(player).GetActivePokemon()

	states := make([]state.StateSnapshot, 0)

	abilityText := fmt.Sprintf("%s activated their ability: %s", playerPokemon.Nickname, playerPokemon.Ability.Name)

	switch playerPokemon.Ability.Name {
	// TEST: no gen 1 pkm have this ability
	case "speed-boost":
		states = append(states, state.NewMessageOnlySnapshot(abilityText))
		states = append(states, state.StatChangeHandler(gameState, playerPokemon, game.StatChange{Change: 1, StatName: "speed"}, 100))
	}

	return states
}

type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		StateUpdates  []state.StateSnapshot
	}
	TurnResolvedMessage struct {
		StateUpdates []state.StateSnapshot
	}
	GameOverMessage struct {
		ForThisPlayer bool
	}
	ContinueUpdaterMessage struct {
		Actions []state.Action
	}
)
