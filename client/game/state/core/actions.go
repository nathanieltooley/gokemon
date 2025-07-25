// Package core (thereafter refered to as state.core to avoid confusion) contains all foundational types and functions for handling game state.
// Like game.core, state.core MUST not rely on other packages.
// The only exception here is game.core which is the foundation's foundation so to speak.
package core

import (
	"github.com/nathanieltooley/gokemon/client/game/core"
)

type Action interface {
	UpdateState(GameState) []StateEvent

	GetCtx() ActionCtx
}

type ActionCtx struct {
	PlayerID int
}

func NewActionCtx(playerID int) ActionCtx {
	return ActionCtx{PlayerID: playerID}
}

type SwitchAction struct {
	Ctx ActionCtx

	SwitchIndex int
	Poke        core.Pokemon
}

func NewSwitchAction(state *GameState, playerID int, switchIndex int) SwitchAction {
	return SwitchAction{
		Ctx: NewActionCtx(playerID),
		// TODO: OOB Check
		SwitchIndex: switchIndex,

		Poke: state.GetPlayer(playerID).Team[switchIndex],
	}
}

func (a SwitchAction) UpdateState(state GameState) []StateEvent {
	return []StateEvent{SwitchEvent{PlayerIndex: a.Ctx.PlayerID, SwitchIndex: a.SwitchIndex}}
}

func (a SwitchAction) GetCtx() ActionCtx {
	return a.Ctx
}

type SkipAction struct {
	Ctx ActionCtx
}

func NewSkipAction(playerId int) SkipAction {
	return SkipAction{
		Ctx: NewActionCtx(playerId),
	}
}

func (a SkipAction) UpdateState(state GameState) []StateEvent {
	return []StateEvent{
		NewMessageEvent("skip turn"),
	}
}

func (a SkipAction) GetCtx() ActionCtx {
	return a.Ctx
}
