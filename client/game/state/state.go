package state

import (
	"fmt"
	"math"
	"slices"
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

func DefaultTeam() []game.Pokemon {
	defaultTeam := make([]game.Pokemon, 0)

	defaultMove := global.MOVES.GetMove("tackle")
	defaultTeam = append(defaultTeam, game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build())
	defaultTeam = append(defaultTeam, game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build())

	defaultTeam[0].Moves[0] = defaultMove

	return defaultTeam
}

func RandomTeam() []game.Pokemon {
	team := make([]game.Pokemon, 6)

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

func NewState(localTeam []game.Pokemon, opposingTeam []game.Pokemon) GameState {
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
		if pokemon.Hp.Value > 0 {
			hostLoss = false
			// log.Debug().Msgf("Host hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
			break
		}
	}

	peerLoss := true
	for _, pokemon := range g.OpposingPlayer.Team {
		if pokemon.Hp.Value > 0 {
			peerLoss = false
			// log.Debug().Msgf("Peer hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
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

// Creates a copy of this state, handling new slice creation and allocation
func (g GameState) Clone() GameState {
	newState := g
	newLTeam := slices.Clone(newState.LocalPlayer.Team)
	newOTeam := slices.Clone(newState.OpposingPlayer.Team)

	newState.LocalPlayer.Team = newLTeam
	newState.OpposingPlayer.Team = newOTeam

	return newState
}

type Player struct {
	Name            string
	Team            []game.Pokemon
	ActivePokeIndex int
}

// TODO: OOB Error handling
func (p Player) GetActivePokemon() game.Pokemon {
	return p.GetPokemon(p.ActivePokeIndex)
}

// Get a copy of a pokemon on a player's team
func (p Player) GetPokemon(index int) game.Pokemon {
	return p.Team[index]
}

func (p *Player) SetPokemon(index int, pokemon game.Pokemon) {
	p.Team[index] = pokemon
}

type ActionCtx struct {
	PlayerId int
}

func NewActionCtx(playerId int) ActionCtx {
	return ActionCtx{PlayerId: playerId}
}

type StateUpdate struct {
	// The resulting state from a given action
	State GameState
	// The messages that communicate what happened
	Messages []string
}

type Action interface {
	// Takes in a state and returns a new state and messages that communicate what happened
	UpdateState(GameState) StateUpdate

	Ctx() ActionCtx
}

type SwitchAction struct {
	ctx ActionCtx

	SwitchIndex int
	Poke        game.Pokemon
}

func NewSwitchAction(state *GameState, playerId int, switchIndex int) *SwitchAction {
	return &SwitchAction{
		ctx: NewActionCtx(playerId),
		// TODO: OOB Check
		SwitchIndex: switchIndex,

		Poke: state.GetPlayer(playerId).Team[switchIndex],
	}
}

func (a *SwitchAction) UpdateState(state GameState) StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	log.Info().Msgf("Player %d: %s, switchs to pokemon %d", a.ctx.PlayerId, playerIntToString(a.ctx.PlayerId), a.SwitchIndex)
	// TODO: OOB Check
	player.ActivePokeIndex = a.SwitchIndex

	messages := []string{fmt.Sprintf("Player %d switched to pokemon %d", a.ctx.PlayerId, a.SwitchIndex)}
	return StateUpdate{
		State:    state,
		Messages: messages,
	}
}

func (a SwitchAction) Ctx() ActionCtx {
	return a.ctx
}

type AttackAction struct {
	ctx ActionCtx

	AttackerMove int

	attackPercent uint
	pokemonName   string
	moveName      string
}

func NewAttackAction(attacker int, attackMove int) *AttackAction {
	return &AttackAction{
		ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

func (a *AttackAction) UpdateState(state GameState) StateUpdate {
	attacker := state.GetPlayer(a.ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	a.pokemonName = attackPokemon.Nickname
	move := attackPokemon.Moves[a.AttackerMove]

	a.moveName = move.Name

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	damage := game.Damage(attackPokemon, defPokemon, move)
	log.Info().Msgf("Player %d: %s attacks %d: %s and deals %d damage", a.ctx.PlayerId, playerIntToString(a.ctx.PlayerId), defenderInt, playerIntToString(defenderInt), damage)

	log.Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
	a.attackPercent = uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

	defPokemon.Hp.Value = defPokemon.Hp.Value - damage

	attacker.SetPokemon(attacker.ActivePokeIndex, attackPokemon)
	defender.SetPokemon(defender.ActivePokeIndex, defPokemon)

	effectiveness := defPokemon.Base.DefenseEffectiveness(game.GetAttackTypeMapping(move.Type))

	log.Debug().Float32("effectiveness", effectiveness).Msg("")

	effectivenessText := ""

	if effectiveness >= 2 {
		effectivenessText = "It was super effective!"
	} else if effectiveness <= 0.5 {
		effectivenessText = "It was not very effective"
	} else if effectiveness == 0 {
		effectivenessText = "It had no effect"
	}

	messages := []string{
		fmt.Sprintf("Player %d's %s used %s", a.ctx.PlayerId, a.pokemonName, a.moveName),
		fmt.Sprintf("It dealt %d%% damage", a.attackPercent),
	}

	if effectivenessText != "" {
		messages = append(messages, effectivenessText)
	}

	return StateUpdate{
		State:    state,
		Messages: messages,
	}
}

func (a AttackAction) Ctx() ActionCtx {
	return a.ctx
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

func (a *SkipAction) UpdateState(state GameState) StateUpdate {
	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("Player %d skipped their turn", a.ctx.PlayerId)},
	}
}

func (a SkipAction) Ctx() ActionCtx {
	return a.ctx
}
