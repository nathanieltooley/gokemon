package ai

import "github.com/nathanieltooley/gokemon/client/game/state"

func BestAction(gameState *state.GameState) state.Action {
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
