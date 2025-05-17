package state

import (
	"math/rand"

	"github.com/nathanieltooley/gokemon/client/game"
)

// Determines the best AI Action. Failsafes to skip action
func BestAiAction(gameState *GameState) Action {
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
			return &SkipAction{}
		}

		bestMoveIndex := -1

		if aiPokemon.Speed() < playerPokemon.Speed() {
			bestMoveIndex = bestSlowingMove(gameState)
		} else {
			bestMoveIndex = bestAttackingMove(gameState)
		}

		bestMove := game.Move{}
		if bestMoveIndex != -1 && bestMoveIndex < 4 {
			bestMove = aiPokemon.Moves[bestMoveIndex]
		}

		if bestMove.IsNil() {
			// Randomly select a non-nil move if no best move available
			for {
				rMoveIndex := rand.Intn(4)
				randMove := aiPokemon.Moves[rMoveIndex]
				if !randMove.IsNil() {
					return NewAttackAction(AI, rMoveIndex)
				}
			}
		} else {
			return NewAttackAction(AI, bestMoveIndex)
		}

	} else {
		// Switch on death
		for i, pokemon := range gameState.ClientPlayer.Team {
			if pokemon.Alive() {
				return NewSwitchAction(gameState, AI, i)
			}
		}
	}

	return &SkipAction{}
}

func bestAttackingMove(gameState *GameState) int {
	aiPokemon := gameState.ClientPlayer.GetActivePokemon()
	playerPokemon := gameState.HostPlayer.GetActivePokemon()

	bestMoveIndex := -1
	var bestMoveDamage uint = 0

	for i, move := range aiPokemon.Moves {
		if move.IsNil() {
			continue
		}

		// assume no crits
		moveDamage := Damage(*aiPokemon, *playerPokemon, move, false, gameState.Weather)
		if moveDamage > bestMoveDamage {
			bestMoveIndex = i
			bestMoveDamage = moveDamage
		}
	}

	return bestMoveIndex
}

func bestSlowingMove(gameState *GameState) int {
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
			if statChange.StatName == STAT_SPEED {
				moveCanSlow = true
			}
		}

		if moveCanSlow {
			chance := move.Accuracy
			if chance > bestSlowChance {
				bestMove = i
			}
		} else if move.Meta.Ailment.Name == "paralysis" && playerPokemon.Status == game.STATUS_NONE { // we make sure the player's pokemon can be para'd
			chance := move.Meta.AilmentChance
			if chance > bestSlowChance {
				bestMove = i
			}
		}
	}

	return bestMove
}
