package state

import (
	"fmt"
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
	Weather        int

	MessageHistory []string
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
	// Make sure pokemon are inited correctly
	for i, p := range localTeam {
		p.CanAttackThisTurn = true

		for i, m := range p.Moves {
			if m != nil {
				p.InGameMoveInfo[i] = game.BattleMove{
					Info: m,
					PP:   uint(m.PP),
				}
			}
		}

		localTeam[i] = p
	}

	for i, p := range opposingTeam {
		p.CanAttackThisTurn = true

		for i, m := range p.Moves {
			if m != nil {
				p.InGameMoveInfo[i] = game.BattleMove{
					Info: m,
					PP:   uint(m.PP),
				}
			}
		}

		opposingTeam[i] = p
	}

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
			log.Debug().Msgf("Host hasn't lost yet, still has pokemon: %s", pokemon.Nickname)
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
func (p Player) GetActivePokemon() *game.Pokemon {
	return p.GetPokemon(p.ActivePokeIndex)
}

// Get a copy of a pokemon on a player's team
func (p Player) GetPokemon(index int) *game.Pokemon {
	return &p.Team[index]
}

type ActionCtx struct {
	PlayerId int
}

func NewActionCtx(playerId int) ActionCtx {
	return ActionCtx{PlayerId: playerId}
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

type Action interface {
	// Takes in a state and returns a new state and messages that communicate what happened
	UpdateState(GameState) []StateSnapshot

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

func (a *SwitchAction) UpdateState(state GameState) []StateSnapshot {
	player := state.GetPlayer(a.ctx.PlayerId)
	log.Info().Msgf("Player %d: %s, switchs to pokemon %d", a.ctx.PlayerId, playerIntToString(a.ctx.PlayerId), a.SwitchIndex)
	// TODO: OOB Check
	player.ActivePokeIndex = a.SwitchIndex

	states := make([]StateSnapshot, 0)

	newActivePkm := player.GetActivePokemon()

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == game.STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
		log.Info().Msgf("%s had their toxic count reset to 1", newActivePkm.Nickname)
	}

	// --- Activate Abilities
	switch newActivePkm.Ability.Name {
	case "drizzle":
		state.Weather = game.WEATHER_RAIN
		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "intimidate":
		opPokemon := state.GetPlayer(invertPlayerIndex(a.ctx.PlayerId)).GetActivePokemon()
		if opPokemon.Ability.Name != "oblivious" && opPokemon.Ability.Name != "own-tempo" && opPokemon.Ability.Name != "inner-focus" {
			opPokemon.Attack.DecreaseStage(1)
		}

		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "natural-cure":
		newActivePkm.Status = game.STATUS_NONE
		states = append(states, NewStateSnapshot(&state, newActivePkm.AbilityText()))
	case "pressure":
		states = append(states, NewMessageOnlySnapshot(fmt.Sprintf("%s is exerting pressure!", newActivePkm.Nickname)))
	}

	newActivePkm.SwitchedInThisTurn = true

	messages := []string{fmt.Sprintf("Player %d switched to %s", a.ctx.PlayerId, newActivePkm.Nickname)}
	states = append(states, StateSnapshot{State: state, Messages: messages})
	return states
}

func (a SwitchAction) Ctx() ActionCtx {
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

func (a *SkipAction) UpdateState(state GameState) []StateSnapshot {
	return []StateSnapshot{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("Player %d skipped their turn", a.ctx.PlayerId)},
		},
	}
}

func (a SkipAction) Ctx() ActionCtx {
	return a.ctx
}
