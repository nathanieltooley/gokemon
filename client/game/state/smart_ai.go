package state

import (
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog/log"
)

// Determines the best AI Action. Failsafes to skip action
func BestAiAction(gameState *stateCore.GameState) stateCore.Action {
	if gameState.ClientPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.HostPlayer.GetActivePokemon()
		aiPokemon := gameState.ClientPlayer.GetActivePokemon()

		hasAnyMoves := false
		for _, move := range aiPokemon.Moves {
			if !move.IsNil() {
				hasAnyMoves = true
				break
			}
		}

		if !hasAnyMoves {
			return &stateCore.SkipAction{}
		}

		bestMoveIndex := -1

		if aiPokemon.Speed(gameState.Weather) < playerPokemon.Speed(gameState.Weather) {
			bestMoveIndex = bestSlowingMove(gameState)
		} else {
			bestMoveIndex = bestAttackingMove(gameState)
		}

		if bestMoveIndex == -1 {
			log.Warn().Msgf("pokemon %s has no moves and / or is dead and should not be here in the first place", aiPokemon.Nickname)
			return &stateCore.SkipAction{}
		}

		bestMove := core.Move{}
		if bestMoveIndex != -1 && bestMoveIndex < 4 {
			bestMove = aiPokemon.Moves[bestMoveIndex]
		}

		if bestMove.IsNil() {
			// Randomly select a non-nil move if no best move available
			for {
				rMoveIndex := global.GokeRand.IntN(4)
				randMove := aiPokemon.Moves[rMoveIndex]
				if !randMove.IsNil() {
					return stateCore.NewAttackAction(stateCore.AI, rMoveIndex)
				}
			}
		} else {
			return stateCore.NewAttackAction(stateCore.AI, bestMoveIndex)
		}

	} else {
		// Switch on death
		for i, pokemon := range gameState.ClientPlayer.Team {
			if pokemon.Alive() {
				return stateCore.NewSwitchAction(gameState, stateCore.AI, i)
			}
		}
	}

	return &stateCore.SkipAction{}
}

func bestAttackingMove(gameState *stateCore.GameState) int {
	aiPokemon := gameState.ClientPlayer.GetActivePokemon()
	playerPokemon := gameState.HostPlayer.GetActivePokemon()

	bestMoveIndex := -1
	var bestMoveDamage uint = 0

	for i, move := range aiPokemon.Moves {
		if move.IsNil() {
			continue
		}

		// assume no crits
		moveDamage := stateCore.Damage(*aiPokemon, *playerPokemon, move, false, gameState.Weather, global.GokeRand)
		if moveDamage > bestMoveDamage {
			bestMoveIndex = i
			bestMoveDamage = moveDamage
		}
	}

	return bestMoveIndex
}

func bestSlowingMove(gameState *stateCore.GameState) int {
	aiPokemon := gameState.ClientPlayer.GetActivePokemon()
	playerPokemon := gameState.HostPlayer.GetActivePokemon()

	bestSlowChance := 0
	bestMove := -1

	for i, move := range aiPokemon.Moves {
		if move.IsNil() {
			continue
		}

		moveCanSlow := false
		for _, statChange := range move.StatChanges {
			if statChange.StatName == core.STAT_SPEED {
				moveCanSlow = true
			}
		}

		if moveCanSlow {
			chance := move.Accuracy
			if chance > bestSlowChance {
				bestMove = i
			}
		} else if move.Meta.Ailment.Name == "paralysis" && playerPokemon.Status == core.STATUS_NONE { // we make sure the player's pokemon can be para'd
			chance := move.Meta.AilmentChance
			if chance > bestSlowChance {
				bestMove = i
			}
		}
	}

	return bestMove
}
