// Package core (thereafter refered to as state.core to avoid confusion) contains all foundational types and functions for handling game state.
// Like game.core, state.core MUST not rely on other packages.
// The only exception here is game.core which is the foundation's foundation so to speak.
package core

import (
	"fmt"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/rs/zerolog/log"
)

type Action interface {
	// Takes in a state and returns a new state and messages that communicate what happened
	UpdateState(GameState) []StateSnapshot

	GetCtx() ActionCtx
}

type ActionCtx struct {
	PlayerId int
}

func NewActionCtx(playerId int) ActionCtx {
	return ActionCtx{PlayerId: playerId}
}

type SwitchAction struct {
	Ctx ActionCtx

	SwitchIndex int
	Poke        core.Pokemon
}

func NewSwitchAction(state *GameState, playerId int, switchIndex int) SwitchAction {
	return SwitchAction{
		Ctx: NewActionCtx(playerId),
		// TODO: OOB Check
		SwitchIndex: switchIndex,

		Poke: state.GetPlayer(playerId).Team[switchIndex],
	}
}

func (a SwitchAction) UpdateState(state GameState) []StateSnapshot {
	player := state.GetPlayer(a.Ctx.PlayerId)
	log.Info().Msgf("%s switches to %s", player.Name, a.Poke.Nickname)
	// TODO: OOB Check
	player.ActivePokeIndex = a.SwitchIndex

	states := make([]StateSnapshot, 0)

	newActivePkm := player.GetActivePokemon()

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == core.STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
		log.Info().Msgf("%s had their toxic count reset to 1", newActivePkm.Nickname)
	}

	// --- Activate Abilities
	switch newActivePkm.Ability.Name {
	case "drizzle":
		state.Weather = core.WEATHER_RAIN
		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "intimidate":
		opPokemon := state.GetPlayer(invertPlayerIndex(a.Ctx.PlayerId)).GetActivePokemon()
		if opPokemon.Ability.Name != "oblivious" && opPokemon.Ability.Name != "own-tempo" && opPokemon.Ability.Name != "inner-focus" {
			opPokemon.Attack.DecreaseStage(1)
		}

		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "natural-cure":
		newActivePkm.Status = core.STATUS_NONE
		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "pressure":
		states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s is exerting pressure!", newActivePkm.Nickname)))
	}

	newActivePkm.SwitchedInThisTurn = true

	messages := make([]string, 0)
	if state.Turn == 0 {
		messages = append(messages, fmt.Sprintf("%s sent in %s!", player.Name, newActivePkm.Nickname))
	} else {
		messages = append(messages, fmt.Sprintf("%s switched to %s!", player.Name, newActivePkm.Nickname))
	}
	states = append(states, StateSnapshot{State: state, Messages: messages})
	return states
}

func (a SwitchAction) GetCtx() ActionCtx {
	return a.Ctx
}

func invertPlayerIndex(initial int) int {
	if initial == HOST {
		return PEER
	} else {
		return HOST
	}
}

type SkipAction struct {
	Ctx ActionCtx
}

func NewSkipAction(playerId int) SkipAction {
	return SkipAction{
		Ctx: NewActionCtx(playerId),
	}
}

func (a SkipAction) UpdateState(state GameState) []StateSnapshot {
	return []StateSnapshot{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("Player %d skipped their turn", a.Ctx.PlayerId)},
		},
	}
}

func (a SkipAction) GetCtx() ActionCtx {
	return a.Ctx
}
