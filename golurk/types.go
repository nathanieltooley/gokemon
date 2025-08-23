package golurk

import (
	"math/rand/v2"
	"slices"
)

// plus 1 because Go has made very stupid design decisions
const (
	HOST = iota + 1
	PEER
)

// Renamed HOST and PEER constants
const (
	PLAYER = iota + 1
	AI
)

type GameState struct {
	HostPlayer   Player
	ClientPlayer Player
	Turn         int
	Weather      int
	Networked    bool
	// An RngSource is stored here directly instead of inside an instance of rand.Rand.
	// This helps in the case of multiplayer where no pointers or interfaces need to be sent,
	// the client just creates the rand.Rand struct when they need it
	RngSource rand.PCG

	MessageHistory []string
}

type Player struct {
	Name            string
	Team            []Pokemon
	ActivePokeIndex int

	// Whether the player's active pokemon was ko'ed this turn.
	// This is separate from ActivePokemon.IsAlive() since this should
	// be persistent across the turn and not go away after switch in.
	// That does mean that this needs to be reset every turn
	ActiveKOed bool

	TimerPaused    bool
	MultiTimerTick int64
}

func (p Player) Lost() bool {
	for _, pokemon := range p.Team {
		if pokemon.Alive() {
			internalLogger.V(2).Info("Player hasn't lost yet", "player_name", p.Name, "alive_pokemon_name", pokemon.Name())
			return false
		}
	}

	return true
}

func (g *GameState) TickPlayerTimers() {
	if !g.HostPlayer.TimerPaused {
		g.HostPlayer.MultiTimerTick--
	}

	if !g.ClientPlayer.TimerPaused {
		g.ClientPlayer.MultiTimerTick--
	}
}

func (g *GameState) GetPlayer(index int) *Player {
	if index == HOST {
		return &g.HostPlayer
	} else {
		return &g.ClientPlayer
	}
}

// GameOver returns whether the game should be over (all of a player's pokemon are dead)
// Value will be -1 for no loser yet, or the winner HOST or PEER
func (g *GameState) GameOver() int {
	hostLoss := g.HostPlayer.Lost()
	peerLoss := g.ClientPlayer.Lost()

	if hostLoss {
		return HOST
	}

	if peerLoss {
		return PEER
	}

	return -1
}

// Clone creates a copy of this state, handling new slice creation and allocation
func (g GameState) Clone() GameState {
	newState := g
	newLTeam := slices.Clone(newState.HostPlayer.Team)
	newOTeam := slices.Clone(newState.ClientPlayer.Team)

	newState.HostPlayer.Team = newLTeam
	newState.ClientPlayer.Team = newOTeam

	return newState
}

func (g *GameState) CreateRng() *rand.Rand {
	return rand.New(&g.RngSource)
}

// CreateNewRng creates a new COPY of RngSource such that RNG calls to the copy
// do not affect the original
func (g *GameState) CreateNewRng() rand.Rand {
	return *rand.New(&g.RngSource)
}

func (p Player) GetActivePokemon() *Pokemon {
	return p.GetPokemon(p.ActivePokeIndex)
}

// GetPokemon gets a player's pokemon at some index
func (p Player) GetPokemon(index int) *Pokemon {
	return &p.Team[index]
}

func (p Player) GetAllAlivePokemon() []*Pokemon {
	alivePokemon := make([]*Pokemon, 0)

	for i, pokemon := range p.Team {
		if pokemon.Hp.Value > 0 {
			// grab pointer directly from team slice
			// loop var pokemon should be a copy and thus a pointer would do nothing
			alivePokemon = append(alivePokemon, &p.Team[i])
		}
	}

	return alivePokemon
}
