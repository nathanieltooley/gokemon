package stateupdater

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/state"
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
		return state.SkipAction{}
	} else {
		for i, pokemon := range gameState.OpposingPlayer.Team {
			if pokemon != nil && pokemon.Alive() {
				return state.SwitchAction{
					PlayerIndex: state.AI,
					SwitchIndex: i,
				}
			}
		}
	}

	return state.SkipAction{}
}

func (u LocalUpdater) Update(gameState *state.GameState) tea.Cmd {
	u.AIAction = u.BestAiAction(gameState)

	// for now Host goes first
	u.PlayerAction.UpdateState(gameState)

	// Force the AI to update their action if their pokemon died
	// this will have to be expanded for the player and eventually
	// the opposing player instead of just the AI
	switch u.AIAction.(type) {
	case state.AttackAction, state.SkipAction:
		if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
			u.AIAction = u.BestAiAction(gameState)
		}
	}

	u.AIAction.UpdateState(gameState)

	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		return func() tea.Msg {
			return ForceSwitchMessage{}
		}
	}

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
