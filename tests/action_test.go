package tests

import (
	"testing"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/game/state"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
)

func TestForceSwitch(t *testing.T) {
	playerPokemonTeam := []core.Pokemon{getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon(), getDummyPokemon()}
	enemyPokemon := getDummyPokemon()

	enemyPokemon.Moves[0] = *global.MOVES.GetMove("roar")

	gameState := state.NewState(playerPokemonTeam, []core.Pokemon{enemyPokemon})

	_ = state.ProcessTurn(&gameState, []stateCore.Action{stateCore.NewAttackAction(stateCore.PEER, 0)})

	if gameState.HostPlayer.ActivePokeIndex == 0 {
		t.Fatalf("Pokemon not switched out")
	}
}
