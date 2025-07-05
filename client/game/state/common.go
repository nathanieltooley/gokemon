package state

import (
	"cmp"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

func commonSwitching(gameState *stateCore.GameState, switches []stateCore.SwitchAction) []stateCore.StateSnapshot {
	states := make([]stateCore.StateSnapshot, 0)

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b stateCore.SwitchAction) int {
		return cmp.Compare(a.Poke.Speed(gameState.Weather), b.Poke.Speed(gameState.Weather))
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a stateCore.SwitchAction, i int) {
		states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
	})

	return states
}

func commonOtherActionHandling(gameState *stateCore.GameState, actions []stateCore.Action) []stateCore.StateSnapshot {
	states := make([]stateCore.StateSnapshot, 0)

	// Reset turn flags
	// eventually this will have to change for double battles
	gameState.HostPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.HostPlayer.GetActivePokemon().SwitchedInThisTurn = false

	gameState.ClientPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.ClientPlayer.GetActivePokemon().SwitchedInThisTurn = false

	// Sort Other Actions
	slices.SortFunc(actions, func(a, b stateCore.Action) int {
		var aSpeed int
		var bSpeed int
		var aPriority int
		var bPriority int

		activePokemon := gameState.GetPlayer(a.GetCtx().PlayerId).GetActivePokemon()
		aSpeed = activePokemon.Speed(gameState.Weather)

		switch a := a.(type) {
		case *stateCore.AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case *stateCore.SkipAction:
			aPriority = -100
		default:
			return 0
		}

		activePokemon = gameState.GetPlayer(b.GetCtx().PlayerId).GetActivePokemon()
		bSpeed = activePokemon.Speed(gameState.Weather)

		switch b := b.(type) {
		case *stateCore.AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case *stateCore.SkipAction:
			bPriority = -100
		default:
			return 0
		}

		log.Debug().
			Int("aPlayer", a.GetCtx().PlayerId).
			Int("bPlayer", b.GetCtx().PlayerId).
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
	lo.ForEach(actions, func(a stateCore.Action, i int) {
		switch a.(type) {
		case *stateCore.AttackAction, *stateCore.SkipAction:
			player := gameState.GetPlayer(a.GetCtx().PlayerId)

			log.Info().Int("attackIndex", i).
				Int("attackerSpeed", player.GetActivePokemon().Speed(gameState.Weather)).
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
			if pokemon.Status == core.STATUS_PARA {
				states = append(states, stateCore.ParaHandler(gameState, pokemon))
			}

			// Skip attack with sleep
			if pokemon.Status == core.STATUS_SLEEP {
				states = append(states, stateCore.SleepHandler(gameState, pokemon))
			}

			// Skip attack with frozen
			if pokemon.Status == core.STATUS_FROZEN {
				states = append(states, stateCore.FreezeHandler(gameState, pokemon))
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				states = append(states, stateCore.ConfuseHandler(gameState, pokemon))
				pokemon.ConfusionCount--

				log.Debug().Int("newConfCount", pokemon.ConfusionCount).Msg("confusion turn completed")
			}

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
			}
		default:
			states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
		}
	})

	return states
}

func commonEndOfTurn(gameState *stateCore.GameState) []stateCore.StateSnapshot {
	states := make([]stateCore.StateSnapshot, 0)

	applyBurn := func(pokemon *core.Pokemon) {
		states = append(states, stateCore.BurnHandler(gameState, pokemon))
	}

	applyPoison := func(pokemon *core.Pokemon) {
		states = append(states, stateCore.PoisonHandler(gameState, pokemon))
	}

	applyToxic := func(pokemon *core.Pokemon) {
		states = append(states, stateCore.ToxicHandler(gameState, pokemon))
	}

	localPokemon := gameState.HostPlayer.GetActivePokemon()
	switch localPokemon.Status {
	case core.STATUS_BURN:
		applyBurn(localPokemon)
	case core.STATUS_POISON:
		applyPoison(localPokemon)
	case core.STATUS_TOXIC:
		applyToxic(localPokemon)
	}

	opPokemon := gameState.ClientPlayer.GetActivePokemon()
	switch opPokemon.Status {
	case core.STATUS_BURN:
		applyBurn(opPokemon)
	case core.STATUS_POISON:
		applyPoison(opPokemon)
	case core.STATUS_TOXIC:
		applyToxic(opPokemon)
	}

	if gameState.Weather == core.WEATHER_SANDSTORM {
		states = append(states, stateCore.SandstormDamage(gameState, localPokemon))
		states = append(states, stateCore.SandstormDamage(gameState, opPokemon))
	}

	endOfTurnAbilities(gameState, HOST)
	endOfTurnAbilities(gameState, PEER)

	messages := lo.FlatMap(states, func(item stateCore.StateSnapshot, i int) []string {
		return item.Messages
	})

	log.Info().Msgf("States: %d", len(states))
	log.Info().Strs("Queued Messages", messages).Msg("")

	gameState.MessageHistory = append(gameState.MessageHistory, messages...)

	return states
}
