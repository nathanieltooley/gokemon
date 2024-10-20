package state

import (
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/global"
)

const (
	HOST = iota
	PEER
)

type GameState struct {
	localPlayer    Player
	opposingPlayer Player
	turn           int

	// HOST or PEER
	// The HOST state is the arbiter of truth
	// and the PEER state is the replicated state of the HOST
	stateType int
}

func NewState() GameState {
	// For testing purposes only
	var defaultTeam [6]*game.Pokemon
	defaultTeam[0] = game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
	defaultTeam[1] = game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build()

	localPlayer := Player{
		Name: "Local",
		Team: defaultTeam,
	}
	opposingPlayer := Player{
		Name: "Opponent",
		Team: defaultTeam,
	}

	return GameState{
		localPlayer:    localPlayer,
		opposingPlayer: opposingPlayer,
		turn:           HOST,
		stateType:      HOST,
	}
}

func (g *GameState) GetPlayer(index int) *Player {
	if index == HOST {
		return &g.localPlayer
	} else {
		return &g.opposingPlayer
	}
}

func (g *GameState) Update(action Action) {
	action.UpdateState(g)
}

type Player struct {
	Name            string
	Team            [6]*game.Pokemon
	ActivePokeIndex uint8
}

type Action interface {
	// Updates the state using a pointer, based on what type of action it is
	UpdateState(*GameState)
}

type SwitchAction struct {
	PlayerIndex int
	SwitchIndex int
}

func (a SwitchAction) UpdateState(state *GameState) {
	player := state.GetPlayer(a.PlayerIndex)
	player.ActivePokeIndex = uint8(a.SwitchIndex)
}

type AttackAction struct {
	Attacker     int
	AttackerMove int
}

func (a AttackAction) UpdateState(state *GameState) {
	attacker := state.GetPlayer(a.Attacker)
	defender := state.GetPlayer(invertPlayerIndex(a.Attacker))

	attackPokemon := attacker.Team[attacker.ActivePokeIndex]
	defPokemon := defender.Team[defender.ActivePokeIndex]

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	damage := game.Damage(attackPokemon, defPokemon, attackPokemon.Moves[a.AttackerMove])
	defPokemon.Hp.Value = defPokemon.Hp.Value - int16(damage)
}

func invertPlayerIndex(initial int) int {
	if initial == HOST {
		return PEER
	} else {
		return HOST
	}
}
