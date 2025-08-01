package golurk

// Determines the best AI Action. Failsafes to skip action
func BestAiAction(gameState *GameState) Action {
	if gameState.ClientPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.HostPlayer.GetActivePokemon()
		aiPokemon := gameState.ClientPlayer.GetActivePokemon()

		rngCopy := gameState.CreateNewRng()

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

		if aiPokemon.Speed(gameState.Weather) < playerPokemon.Speed(gameState.Weather) {
			bestMoveIndex = bestSlowingMove(gameState)
		} else {
			bestMoveIndex = bestAttackingMove(gameState)
		}

		if bestMoveIndex == -1 {
			internalLogger.WithName("ai_move_selection").Info("pokemon has no moves and / or is dead and should not be here in the first place", "pokemon_name", aiPokemon.Nickname)
			return &SkipAction{}
		}

		bestMove := Move{}
		if bestMoveIndex != -1 && bestMoveIndex < 4 {
			bestMove = aiPokemon.Moves[bestMoveIndex]
		}

		if bestMove.IsNil() {
			// Randomly select a non-nil move if no best move available
			for {
				rMoveIndex := rngCopy.IntN(4)
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

		rng := gameState.CreateNewRng()

		// assume no crits
		moveDamage := Damage(*aiPokemon, *playerPokemon, move, false, gameState.Weather, &rng)
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
		} else if move.Meta.Ailment.Name == "paralysis" && playerPokemon.Status == STATUS_NONE { // we make sure the player's pokemon can be para'd
			chance := move.Meta.AilmentChance
			if chance > bestSlowChance {
				bestMove = i
			}
		}
	}

	return bestMove
}
