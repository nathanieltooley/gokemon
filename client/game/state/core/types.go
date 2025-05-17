package core

import (
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/rs/zerolog/log"
)

const (
	HOST = iota
	PEER
)

// Renamed HOST and PEER constants
const (
	PLAYER = iota
	AI
)

// TODO: Lower size of this struct? (mainly for networking purposes)
// might ignore just cause there haven't been any noticable issues,
// and this is networked very infrequently, only after the end of turns
type GameState struct {
	HostPlayer   Player
	ClientPlayer Player
	Turn         int
	Weather      int
	Networked    bool

	MessageHistory []string
}

type Player struct {
	Name            string
	Team            []core.Pokemon
	ActivePokeIndex int

	// Whether the player's active pokemon was ko'ed this turn.
	// This is separate from ActivePokemon.IsAlive() since this should
	// be persistent across the turn and not go away after switch in.
	// That does mean that this needs to be reset every turn
	ActiveKOed bool

	TimerPaused    bool
	MultiTimerTick int64
}

type StateSnapshot struct {
	// The resulting state from a given action
	State GameState
	// The messages that communicate what happened
	Messages []string
	// hack for an optional value without using pointers
	Empty bool
	// hack for messages that don't have any direct state changes
	MessagesOnly bool
}

func (s *GameState) TickPlayerTimers() {
	if !s.HostPlayer.TimerPaused {
		s.HostPlayer.MultiTimerTick--
	}

	if !s.ClientPlayer.TimerPaused {
		s.ClientPlayer.MultiTimerTick--
	}
}

func (g *GameState) GetPlayer(index int) *Player {
	if index == HOST {
		return &g.HostPlayer
	} else {
		return &g.ClientPlayer
	}
}

// Returns whether the game should be over (all of a player's pokemon are dead)
// Value will be -1 for no loser yet, or the winner HOST or PEER
func (g *GameState) GameOver() int {
	hostLoss := true
	for _, pokemon := range g.HostPlayer.Team {
		if pokemon.Hp.Value > 0 {
			hostLoss = false
			log.Debug().Msgf("Host hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
			break
		}
	}

	peerLoss := true
	for _, pokemon := range g.ClientPlayer.Team {
		if pokemon.Hp.Value > 0 {
			peerLoss = false
			log.Debug().Msgf("Peer hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
			break
		}
	}

	if hostLoss {
		log.Info().Msg("HOST/Player lost")
		return HOST
	}

	if peerLoss {
		log.Info().Msg("PEER/AI lost")
		return PEER
	}

	return -1
}

// Creates a copy of this state, handling new slice creation and allocation
func (g GameState) Clone() GameState {
	newState := g
	newLTeam := slices.Clone(newState.HostPlayer.Team)
	newOTeam := slices.Clone(newState.ClientPlayer.Team)

	newState.HostPlayer.Team = newLTeam
	newState.ClientPlayer.Team = newOTeam

	return newState
}

// TODO: OOB Error handling
func (p Player) GetActivePokemon() *core.Pokemon {
	return p.GetPokemon(p.ActivePokeIndex)
}

// Get a copy of a pokemon on a player's team
func (p Player) GetPokemon(index int) *core.Pokemon {
	return &p.Team[index]
}

// Create a new StateSnapshot. Takes in a pointer to avoid a second copy
func NewStateSnapshot(state *GameState, messages ...string) StateSnapshot {
	return StateSnapshot{
		State:    state.Clone(),
		Messages: messages,
	}
}

func NewEmptyStateSnapshot() StateSnapshot {
	return StateSnapshot{Empty: true}
}

func NewMessageOnlySnapshot(messages ...string) StateSnapshot {
	return StateSnapshot{
		Messages:     messages,
		MessagesOnly: true,
	}
}
