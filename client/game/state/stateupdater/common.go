package stateupdater

import (
	"cmp"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func commonSwitching(gameState *state.GameState, switches []state.SwitchAction) []state.StateSnapshot {
	states := make([]state.StateSnapshot, 0)

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b state.SwitchAction) int {
		return cmp.Compare(a.Poke.Speed(), b.Poke.Speed())
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a state.SwitchAction, i int) {
		states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
	})

	return states
}

func commonOtherActionHandling(gameState *state.GameState, actions []state.Action) []state.StateSnapshot {
	states := make([]state.StateSnapshot, 0)

	// Reset turn flags
	// eventually this will have to change for double battles
	gameState.LocalPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.LocalPlayer.GetActivePokemon().SwitchedInThisTurn = false

	gameState.OpposingPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.OpposingPlayer.GetActivePokemon().SwitchedInThisTurn = false

	// Sort Other Actions
	slices.SortFunc(actions, func(a, b state.Action) int {
		var aSpeed int
		var bSpeed int
		var aPriority int
		var bPriority int

		activePokemon := gameState.GetPlayer(a.Ctx().PlayerId).GetActivePokemon()
		aSpeed = activePokemon.Speed()

		switch a := a.(type) {
		case *state.AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case *state.SkipAction:
			aPriority = -100
		default:
			return 0
		}

		activePokemon = gameState.GetPlayer(b.Ctx().PlayerId).GetActivePokemon()
		bSpeed = activePokemon.Speed()

		switch b := b.(type) {
		case *state.AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case *state.SkipAction:
			bPriority = -100
		default:
			return 0
		}

		log.Debug().
			Int("aPlayer", a.Ctx().PlayerId).
			Int("bPlayer", b.Ctx().PlayerId).
			Int("aSpeed", aSpeed).
			Int("bSpeed", bSpeed).
			Int("aPriority", aPriority).
			Int("bPriority", bPriority).
			Int("comp", cmp.Compare(aSpeed, bSpeed)).
			Int("compPriority", cmp.Compare(aPriority, bPriority)).
			Msg("sort debug")

		priorComp := cmp.Compare(aPriority, bPriority)
		speedComp := cmp.Compare(aSpeed, bSpeed)

		if priorComp == 0 {
			return speedComp
		} else {
			return priorComp
		}
	})

	// Reverse for desc order
	slices.Reverse(actions)

	// Process otherActions next
	lo.ForEach(actions, func(a state.Action, i int) {
		switch a.(type) {
		case *state.AttackAction, *state.SkipAction:
			player := gameState.GetPlayer(a.Ctx().PlayerId)

			log.Info().Int("attackIndex", i).
				Int("attackerSpeed", player.GetActivePokemon().Speed()).
				Int("attackerRawSpeed", player.GetActivePokemon().RawSpeed.CalcValue()).
				Int("attackerConfCount", player.GetActivePokemon().ConfusionCount).
				Msg("Attack state update")

			pokemon := player.GetActivePokemon()
			if pokemon.CanAttackThisTurn {
				pokemon.CanAttackThisTurn = !pokemon.SwitchedInThisTurn
			}

			if !pokemon.Alive() {
				return
			}

			// Skip attack with para
			if pokemon.Status == game.STATUS_PARA {
				states = append(states, state.ParaHandler(gameState, pokemon))
			}

			// Skip attack with sleep
			if pokemon.Status == game.STATUS_SLEEP {
				states = append(states, state.SleepHandler(gameState, pokemon))
			}

			// Skip attack with frozen
			if pokemon.Status == game.STATUS_FROZEN {
				states = append(states, state.FreezeHandler(gameState, pokemon))
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				states = append(states, state.ConfuseHandler(gameState, pokemon))
				pokemon.ConfusionCount--

				log.Debug().Int("newConfCount", pokemon.ConfusionCount).Msg("confusion turn completed")
			}

			endOfTurnAbilities(gameState, a.Ctx().PlayerId)

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
			}
		default:
			states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
		}
	})

	return states
}
