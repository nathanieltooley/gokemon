package state

import (
	"fmt"
	"math"
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

	MessageHistory []string

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

type ActionCtx struct {
	playerId int
}

func NewActionCtx(playerId int) ActionCtx {
	return ActionCtx{playerId: playerId}
}

type Action interface {
	// Updates the state using a pointer, based on what type of action it is
	// Should be pointer receiver method so that Message can have accurate info to send
	UpdateState(*GameState)
	// Returns a list of human readable messages to been shown to both players
	Message() []string
}

type SwitchAction struct {
	ctx ActionCtx

	SwitchIndex int
}

func NewSwitchAction(playerId int, switchIndex int) *SwitchAction {
	return &SwitchAction{
		ctx:         NewActionCtx(playerId),
		SwitchIndex: switchIndex,
	}
}

func (a *SwitchAction) UpdateState(state *GameState) {
	player := state.GetPlayer(a.ctx.playerId)
	log.Info().Msgf("Player %d: %s, switchs to pokemon %d", a.ctx.playerId, playerIntToString(a.ctx.playerId), a.SwitchIndex)
	player.ActivePokeIndex = uint8(a.SwitchIndex)
}

func (a SwitchAction) Message() []string {
	return []string{fmt.Sprintf("Player %d switched to pokemon %d", a.ctx.playerId, a.SwitchIndex)}
}

type AttackAction struct {
	ctx ActionCtx

	AttackerMove int

	attackPercent uint
	pokemonName   string
	moveName      string
	effectiveness string
}

func NewAttackAction(attacker int, attackMove int) *AttackAction {
	return &AttackAction{
		ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

func (a *AttackAction) UpdateState(state *GameState) {
	attacker := state.GetPlayer(a.ctx.playerId)
	defenderInt := invertPlayerIndex(a.ctx.playerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.Team[attacker.ActivePokeIndex]
	defPokemon := defender.Team[defender.ActivePokeIndex]

	a.pokemonName = attackPokemon.Nickname
	move := attackPokemon.Moves[a.AttackerMove]

	a.moveName = move.Name

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	damage := game.Damage(attackPokemon, defPokemon, move)
	log.Info().Msgf("Player %d: %s attacks %d: %s and deals %d damage", a.ctx.playerId, playerIntToString(a.ctx.playerId), defenderInt, playerIntToString(defenderInt), damage)

	log.Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	a.attackPercent = uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	defPokemon.Hp.Value = defPokemon.Hp.Value - int16(damage)
}

func (a AttackAction) Message() []string {
	return []string{
		fmt.Sprintf("Player %d's %s used %s", a.ctx.playerId, a.pokemonName, a.moveName),
		fmt.Sprintf("It dealt %d%% damage", a.attackPercent),
	}
}

func invertPlayerIndex(initial int) int {
	if initial == HOST {
		return PEER
	} else {
		return HOST
	}
}

type SkipAction struct {
	ctx ActionCtx
}

func NewSkipAction(playerId int) *SkipAction {
	return &SkipAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a *SkipAction) UpdateState(state *GameState) { return }
func (a SkipAction) Message() []string {
	return []string{fmt.Sprintf("Player %d skipped their turn", a.ctx.playerId)}
}
