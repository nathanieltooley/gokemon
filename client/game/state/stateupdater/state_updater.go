package stateupdater

import (
	"cmp"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type StateUpdater interface {
	// Updates the state of the game
	Update(*state.GameState, bool) tea.Cmd

	// Sets the Host's action for this turn
	SendAction(state.Action)
}

func syncState(mainState *state.GameState, newState state.StateUpdate) state.StateUpdate {
	// Clone here because of slices
	*mainState = newState.State.Clone()

	return newState
}

type LocalUpdater struct {
	Actions []state.Action
}

func (u *LocalUpdater) BestAiAction(gameState *state.GameState) state.Action {
	if gameState.OpposingPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.LocalPlayer.GetActivePokemon()
		aiPokemon := gameState.OpposingPlayer.GetActivePokemon()

		bestMoveIndex := 0
		var bestMove *game.Move
		var bestMoveDamage uint = 0

		for i, move := range aiPokemon.Moves {
			if move == nil {
				continue
			}

			moveDamage := game.Damage(aiPokemon, playerPokemon, move)
			if moveDamage > bestMoveDamage {
				bestMoveIndex = i
				bestMove = move
				bestMoveDamage = moveDamage
			}
		}

		if bestMove == nil {
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

func (u LocalUpdater) Update(gameState *state.GameState, resolvingForcedSwitches bool) tea.Cmd {
	// FIX: State updates happen before the sending of the TurnResolvedMessage
	u.Actions = append(u.Actions, u.BestAiAction(gameState))

	switches := make([]state.SwitchAction, 0)
	otherActions := make([]state.Action, 0)

	states := make([]state.StateUpdate, 0)

	for _, a := range u.Actions {
		switch a := a.(type) {
		case *state.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b state.SwitchAction) int {
		return cmp.Compare(a.Poke.Speed.Value, b.Poke.Speed.Value)
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a state.SwitchAction, i int) {
		states = append(states, syncState(gameState, a.UpdateState(*gameState)))
	})

	if resolvingForcedSwitches {
		u.Actions = make([]state.Action, 0)

		gameState.Turn++

		return func() tea.Msg {
			time.Sleep(time.Second * 1)

			messages := lo.FlatMap(states, func(item state.StateUpdate, i int) []string {
				return item.Messages
			})

			log.Info().Msgf("States: %d", len(states))
			log.Info().Strs("Queued Messages", messages).Msg("")

			gameState.MessageHistory = append(gameState.MessageHistory, messages...)

			return TurnResolvedMessage{}
		}
	}

	// Sort Other Actions
	slices.SortFunc(otherActions, func(a, b state.Action) int {
		var aSpeed int16
		var bSpeed int16

		switch a := a.(type) {
		case *state.AttackAction:
			aSpeed = gameState.GetPlayer(a.Ctx().PlayerId).GetActivePokemon().Speed.Value
		default:
			return 0
		}

		switch b := a.(type) {
		case *state.AttackAction:
			bSpeed = gameState.GetPlayer(b.Ctx().PlayerId).GetActivePokemon().Speed.Value
		default:
			return 0
		}

		return cmp.Compare(aSpeed, bSpeed)
	})

	// Reverse for desc order
	slices.Reverse(otherActions)

	// Process otherActions next
	lo.ForEach(otherActions, func(a state.Action, i int) {
		switch a.(type) {
		// Only Update if the pokemon is alive
		case *state.AttackAction:
			player := gameState.GetPlayer(a.Ctx().PlayerId)

			if player.GetActivePokemon().Alive() {
				states = append(states, syncState(gameState, a.UpdateState(*gameState)))
			}
		default:
			states = append(states, syncState(gameState, a.UpdateState(*gameState)))
		}
	})

	u.Actions = make([]state.Action, 0)

	// Seems weird but should make sense if or when multiplayer is added
	// also these checks will have to change if double battles are added
	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		return func() tea.Msg {
			return ForceSwitchMessage{
				ForThisPlayer: true,
				StateUpdates:  states,
			}
		}
	}

	if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
		return func() tea.Msg {
			return ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  states,
			}
		}
	}

	messages := lo.FlatMap(states, func(item state.StateUpdate, i int) []string {
		return item.Messages
	})

	log.Info().Msgf("States: %d", len(states))
	log.Info().Strs("Queued Messages", messages).Msg("")

	gameState.MessageHistory = append(gameState.MessageHistory, messages...)

	return func() tea.Msg {
		// Artifical Delay
		time.Sleep(time.Second * 2)

		gameState.Turn++

		return TurnResolvedMessage{
			StateUpdates: states,
		}
	}
}

func (u *LocalUpdater) SendAction(action state.Action) {
	u.Actions = append(u.Actions, action)
}

type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		StateUpdates  []state.StateUpdate
	}
	TurnResolvedMessage struct {
		StateUpdates []state.StateUpdate
	}
)
