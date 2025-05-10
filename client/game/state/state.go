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

func (s *GameState) TickPlayerTimers() {
	if !s.HostPlayer.TimerPaused {
		s.HostPlayer.MultiTimerTick--
	}

	if !s.ClientPlayer.TimerPaused {
		s.ClientPlayer.MultiTimerTick--
	}
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

	defaultTeam[0].Moves[0] = *defaultMove

	return defaultTeam
}

func RandomTeam() []game.Pokemon {
	team := make([]game.Pokemon, 6)

	for i := range 6 {
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
			if !m.IsNil() {
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
			if !m.IsNil() {
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
		HostPlayer:   localPlayer,
		ClientPlayer: opposingPlayer,
		Turn:         0,
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
	newLTeam := slices.Clone(newState.HostPlayer.Team)
	newOTeam := slices.Clone(newState.ClientPlayer.Team)

	newState.HostPlayer.Team = newLTeam
	newState.ClientPlayer.Team = newOTeam

	return newState
}

type Player struct {
	Name            string
	Team            []game.Pokemon
	ActivePokeIndex int

	// Whether the player's active pokemon was ko'ed this turn.
	// This is separate from ActivePokemon.IsAlive() since this should
	// be persistent across the turn and not go away after switch in.
	// That does mean that this needs to be reset every turn
	ActiveKOed bool

	TimerPaused    bool
	MultiTimerTick int64
}

// TODO: OOB Error handling
func (p Player) GetActivePokemon() *game.Pokemon {
	return p.GetPokemon(p.ActivePokeIndex)
}

// Get a copy of a pokemon on a player's team
func (p Player) GetPokemon(index int) *game.Pokemon {
	return &p.Team[index]
}

func GetTimerString(timer int64) string {
	timerInSeconds := timer / int64(global.GameTicksPerSecond)
	minutes := timerInSeconds / 60
	seconds := timerInSeconds % 60

	// there could be a way to do this using a format string
	// but this is easier
	secondsString := fmt.Sprint(seconds)
	if seconds < 10 {
		secondsString = "0" + secondsString
	}

	return fmt.Sprintf("%d:%s", minutes, secondsString)
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

	GetCtx() ActionCtx
}

type SwitchAction struct {
	Ctx ActionCtx

	SwitchIndex int
	Poke        game.Pokemon
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
		opPokemon := state.GetPlayer(invertPlayerIndex(a.Ctx.PlayerId)).GetActivePokemon()
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
