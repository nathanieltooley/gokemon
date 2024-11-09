package stateupdater

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
)

type StateUpdater interface {
	// Updates the state of the game
	Update(*state.GameState) tea.Cmd

	// Sets the Host's action for this turn
	SendAction(state.Action)
}

type LocalUpdater struct {
	PlayerAction state.Action
	AIAction     state.Action
}

func (u *LocalUpdater) BestAiAction(gameState *state.GameState) state.Action {
	if gameState.OpposingPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.LocalPlayer.GetActivePokemon()
		aiPokemon := gameState.OpposingPlayer.GetActivePokemon()

		bestMoveIndex := 0
		var bestMove *game.MoveFull
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
			if pokemon != nil && pokemon.Alive() {
				return state.NewSwitchAction(state.AI, i)
			}
		}
	}

	return &state.SkipAction{}
}

func (u LocalUpdater) Update(gameState *state.GameState) tea.Cmd {
	u.AIAction = u.BestAiAction(gameState)

	// for now Host goes first
	u.PlayerAction.UpdateState(gameState)

	// Force the AI to update their action if their pokemon died
	// this will have to be expanded for the player and eventually
	// the opposing player instead of just the AI
	switch u.AIAction.(type) {
	case *state.AttackAction, *state.SkipAction:
		if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
			u.AIAction = u.BestAiAction(gameState)
		}
	}

	u.AIAction.UpdateState(gameState)

	log.Debug().Msg("Updated state from both actions")

	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		return func() tea.Msg {
			return ForceSwitchMessage{}
		}
	}

	gameState.MessageQueue = append(gameState.MessageQueue, u.PlayerAction.Message()...)
	gameState.MessageQueue = append(gameState.MessageQueue, u.AIAction.Message()...)
	log.Info().Msgf("Queued Message: %s", u.PlayerAction.Message())
	log.Info().Msgf("Queued Message: %s", u.AIAction.Message())

	gameState.MessageHistory = append(gameState.MessageHistory, u.PlayerAction.Message()...)
	gameState.MessageHistory = append(gameState.MessageHistory, u.AIAction.Message()...)

	u.AIAction = nil
	u.PlayerAction = nil
	return func() tea.Msg {
		return TurnResolvedMessage{}
	}
}

func (u *LocalUpdater) SendAction(action state.Action) {
	u.PlayerAction = action
}

type (
	ForceSwitchMessage  struct{}
	TurnResolvedMessage struct{}
)
