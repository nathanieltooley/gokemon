package state

import (
	"fmt"
	"math/rand"
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

type StateUpdate struct {
	// The resulting state from a given action
	State GameState
	// The messages that communicate what happened
	Messages []string
	Empty    bool
}

func NewEmptyStateUpdate() StateUpdate {
	return StateUpdate{Empty: true}
}

type Action interface {
	// Takes in a state and returns a new state and messages that communicate what happened
	UpdateState(GameState) []StateUpdate

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

func (a *SwitchAction) UpdateState(state GameState) []StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	log.Info().Msgf("Player %d: %s, switchs to pokemon %d", a.ctx.PlayerId, playerIntToString(a.ctx.PlayerId), a.SwitchIndex)
	// TODO: OOB Check
	player.ActivePokeIndex = a.SwitchIndex

	newActivePkm := player.GetActivePokemon()

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == game.STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
	}

	newActivePkm.SwitchedInThisTurn = true

	messages := []string{fmt.Sprintf("Player %d switched to pokemon %d", a.ctx.PlayerId, a.SwitchIndex)}
	return []StateUpdate{
		{
			State:    state,
			Messages: messages,
		},
	}
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

func (a *SkipAction) UpdateState(state GameState) []StateUpdate {
	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("Player %d skipped their turn", a.ctx.PlayerId)},
		},
	}
}

func (a SkipAction) Ctx() ActionCtx {
	return a.ctx
}

type SleepAction struct {
	ctx ActionCtx
}

func NewSleepAction(playerId int) *SleepAction {
	return &SleepAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a SleepAction) UpdateState(state GameState) []StateUpdate {
	pokemon := state.GetPlayer(a.ctx.PlayerId).GetActivePokemon()

	message := ""

	// Sleep is over
	// TODO: Add message for waking up
	if pokemon.SleepCount <= 0 {
		pokemon.Status = game.STATUS_NONE
		message = fmt.Sprintf("%s woke up!", pokemon.Nickname)
		pokemon.CanAttackThisTurn = true
	} else {
		message = fmt.Sprintf("%s is asleep", pokemon.Nickname)
		pokemon.CanAttackThisTurn = false
	}

	pokemon.SleepCount--

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{message},
		},
	}
}

type ParaAction struct {
	ctx ActionCtx
}

func NewParaAction(playerId int) *ParaAction {
	return &ParaAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a ParaAction) UpdateState(state GameState) []StateUpdate {
	pokemon := state.GetPlayer(a.ctx.PlayerId).GetActivePokemon()

	paraChance := 0.5
	paraCheck := rand.Float64()

	if paraCheck > paraChance {
		// don't get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check passed")
		return []StateUpdate{NewEmptyStateUpdate()}
	} else {
		// do get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check failed")
		pokemon.CanAttackThisTurn = false
	}

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("Player %d's pokemon is paralyzed and cannot move", a.ctx.PlayerId)},
		},
	}
}

type BurnAction struct {
	ctx ActionCtx
}

func NewBurnAction(playerId int) *BurnAction {
	return &BurnAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a BurnAction) UpdateState(state GameState) []StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	damage := pokemon.MaxHp / 16
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("%s pokemon is burned", pokemon.Nickname), fmt.Sprintf("Burn did %d%% damage", damagePercent)},
		},
	}
}

type PoisonAction struct {
	ctx ActionCtx
}

func NewPoisonAction(playerId int) *PoisonAction {
	return &PoisonAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a PoisonAction) UpdateState(state GameState) []StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	// for future reference, this is MaxHp / 16 in gen 1
	damage := pokemon.MaxHp / 8
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("%s is poisoned", pokemon.Nickname), fmt.Sprintf("Poison did %d%% damage", damagePercent)},
		},
	}
}

type ToxicAction struct {
	ctx ActionCtx
}

func NewToxicAction(playerId int) *ToxicAction {
	return &ToxicAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a ToxicAction) UpdateState(state GameState) []StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	damage := (pokemon.MaxHp / 16) * uint(pokemon.ToxicCount)
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	log.Info().Int("toxicCount", pokemon.ToxicCount).Uint("damage", damage).Msg("Toxic Action")

	pokemon.ToxicCount++

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("%s is badly poisoned", pokemon.Nickname), fmt.Sprintf("Toxic did %d%% damage", damagePercent)},
		},
	}
}

type FrozenAction struct {
	ctx ActionCtx
}

func NewFrozenAction(playerId int) *FrozenAction {
	return &FrozenAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a FrozenAction) UpdateState(state GameState) []StateUpdate {
	pokemon := state.GetPlayer(a.ctx.PlayerId).GetActivePokemon()

	thawChance := .20
	thawCheck := rand.Float64()

	message := ""

	// pokemon stays frozen
	if thawCheck > thawChance {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check failed")
		message = fmt.Sprintf("%s is frozen and cannot move", pokemon.Nickname)

		pokemon.CanAttackThisTurn = false
	} else {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check succeeded!")
		message = fmt.Sprintf("%s thawed out!", pokemon.Nickname)

		pokemon.Status = game.STATUS_NONE
		pokemon.CanAttackThisTurn = true
	}

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{message},
		},
	}
}

type ConfusionAction struct {
	ctx ActionCtx
}

func NewConfusionAction(playerId int) *ConfusionAction {
	return &ConfusionAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a ConfusionAction) UpdateState(state GameState) []StateUpdate {
	confusedPokemon := state.GetPlayer(a.ctx.PlayerId).GetActivePokemon()

	confusedPokemon.ConfusionCount--
	log.Debug().Int("newConfCount", confusedPokemon.ConfusionCount).Msg("confusion lowered")

	confChance := .33
	confCheck := rand.Float64()

	// Exit early
	if confCheck > confChance {
		return []StateUpdate{NewEmptyStateUpdate()}
	}

	confMove := game.Move{
		Name:  "Confusion",
		Power: 40,
		Meta: &game.MoveMeta{
			Category: struct {
				Id   int
				Name string
			}{
				Name: "damage",
			},
		},
		DamageClass: game.DAMAGETYPE_PHYSICAL,
	}

	dmg := game.Damage(*confusedPokemon, *confusedPokemon, &confMove, false)
	confusedPokemon.Damage(dmg)
	confusedPokemon.CanAttackThisTurn = false

	log.Info().Uint("damage", dmg).Msgf("%s hit itself in confusion", confusedPokemon.Nickname)

	return []StateUpdate{
		{
			State:    state,
			Messages: []string{fmt.Sprintf("%s hurt itself in confusion", confusedPokemon.Nickname)},
		},
	}
}

func (a ConfusionAction) Ctx() ActionCtx { return a.ctx }
