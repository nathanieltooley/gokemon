package state

import (
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/global"
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

type GameState struct {
	LocalPlayer    Player
	OpposingPlayer Player
	Turn           int

	// HOST or PEER
	// The HOST state is the arbiter of truth
	// and the PEER state is the replicated state of the HOST
	stateType int
}

func playerIntToString(player int) string {
	switch player {
	case HOST:
		return "Host/Player"
	case PEER:
		return "Peer/AI"
	}

	return ""
}

func DefaultTeam() [6]*game.Pokemon {
	var defaultTeam [6]*game.Pokemon

	defaultMove := global.MOVES.GetMove("tackle")
	defaultTeam[0] = game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build()
	defaultTeam[1] = game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build()

	defaultTeam[0].Moves[0] = defaultMove

	return defaultTeam
}

func RandomTeam() [6]*game.Pokemon {
	var team [6]*game.Pokemon

	for i := 0; i < 6; i++ {
		rndBasePkm := global.POKEMON.GetRandomPokemon()
		rndPkm := game.NewPokeBuilder(rndBasePkm).
			SetRandomEvs().
			SetRandomIvs().
			SetRandomLevel(80, 100).
			SetRandomNature().
			SetRandomMoves(global.MOVES.GetFullMovesForPokemon(rndBasePkm.Name)).
			SetRandomAbility(global.ABILITIES[strings.ToLower(rndBasePkm.Name)]).
			Build()
		team[i] = rndPkm
	}

	return team
}

func NewState(localTeam [6]*game.Pokemon, opposingTeam [6]*game.Pokemon) GameState {
	// For testing purposes only
	localPlayer := Player{
		Name: "Local",
		Team: localTeam,
	}
	opposingPlayer := Player{
		Name: "Opponent",
		Team: opposingTeam,
	}

	return GameState{
		LocalPlayer:    localPlayer,
		OpposingPlayer: opposingPlayer,
		Turn:           0,

		stateType: HOST,
	}
}

func (g *GameState) GetPlayer(index int) *Player {
	if index == HOST {
		return &g.LocalPlayer
	} else {
		return &g.OpposingPlayer
	}
}

// Returns whether the game should be over (all of a player's pokemon are dead)
// Value will be -1 for no loser yet, or the winner HOST or PEER
func (g *GameState) GameOver() int {
	hostLoss := true
	for _, pokemon := range g.LocalPlayer.Team {
		if pokemon == nil {
			continue
		}

		if pokemon.Hp.Value > 0 {
			hostLoss = false
			log.Debug().Msgf("Host hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
			break
		}
	}

	peerLoss := true
	for _, pokemon := range g.OpposingPlayer.Team {
		if pokemon == nil {
			continue
		}

		if pokemon.Hp.Value > 0 {
			peerLoss = false
			log.Debug().Msgf("Peer hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
			break
		}
	}

	if hostLoss {
		log.Info().Msg("HOST/Player lost")
		return PEER
	}

	if peerLoss {
		log.Info().Msg("PEER/AI lost")
		return HOST
	}

	return -1
}

type Player struct {
	Name            string
	Team            [6]*game.Pokemon
	ActivePokeIndex uint8
}

func (p Player) GetActivePokemon() *game.Pokemon {
	return p.Team[p.ActivePokeIndex]
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
	log.Info().Msgf("Player %d: %s, switchs to pokemon %d", a.PlayerIndex, playerIntToString(a.PlayerIndex), a.SwitchIndex)
	player.ActivePokeIndex = uint8(a.SwitchIndex)
}

type AttackAction struct {
	Attacker     int
	AttackerMove int
}

func NewAttackAction(attacker int, attackMove int) AttackAction {
	return AttackAction{
		Attacker:     attacker,
		AttackerMove: attackMove,
	}
}

func (a AttackAction) UpdateState(state *GameState) {
	attacker := state.GetPlayer(a.Attacker)
	defenderInt := invertPlayerIndex(a.Attacker)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.Team[attacker.ActivePokeIndex]
	defPokemon := defender.Team[defender.ActivePokeIndex]

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	damage := game.Damage(attackPokemon, defPokemon, attackPokemon.Moves[a.AttackerMove])
	log.Info().Msgf("Player %d: %s attacks %d: %s and deals %d damage", a.Attacker, playerIntToString(a.Attacker), defenderInt, playerIntToString(defenderInt), damage)
	defPokemon.Hp.Value = defPokemon.Hp.Value - int16(damage)
}

func invertPlayerIndex(initial int) int {
	if initial == HOST {
		return PEER
	} else {
		return HOST
	}
}

type SkipAction struct{}

func (a SkipAction) UpdateState(state *GameState) { return }
