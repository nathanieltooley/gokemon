package state

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
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

	newActivePkm := player.GetActivePokemon()

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == game.STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
	}

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
}

func NewAttackAction(attacker int, attackMove int) *AttackAction {
	return &AttackAction{
		ctx:          NewActionCtx(attacker),
		AttackerMove: attackMove,
	}
}

var attackActionLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "attack-action").Logger()
	return &logger
}

func (a *AttackAction) UpdateState(state GameState) StateUpdate {
	attacker := state.GetPlayer(a.ctx.PlayerId)
	defenderInt := invertPlayerIndex(a.ctx.PlayerId)
	defender := state.GetPlayer(defenderInt)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	// TODO: Make sure a.AttackerMove is between 0 -> 3
	move := attackPokemon.Moves[a.AttackerMove]

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s used %s", attackPokemon.Nickname, move.Name))

	accuracyCheck := rand.Intn(100)
	if accuracyCheck < move.Accuracy {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check passed")
		damage := game.Damage(*attackPokemon, *defPokemon, move)
		log.Info().Msgf("Player %d: %s attacks %d: %s and deals %d damage", a.ctx.PlayerId, playerIntToString(a.ctx.PlayerId), defenderInt, playerIntToString(defenderInt), damage)

		attackActionLogger().Debug().Msgf("Max Hp: %d", defPokemon.MaxHp)
		attackPercent := uint(math.Min(100, (float64(damage)/float64(defPokemon.MaxHp))*100))

		defPokemon.Damage(damage)

		effectiveness := defPokemon.Base.DefenseEffectiveness(game.GetAttackTypeMapping(move.Type))

		attackActionLogger().Debug().Float32("effectiveness", effectiveness).Msg("")

		effectivenessText := ""

		if effectiveness >= 2 {
			effectivenessText = "It was super effective!"
		} else if effectiveness <= 0.5 {
			effectivenessText = "It was not very effective"
		} else if effectiveness == 0 {
			effectivenessText = "It had no effect"
		}

		messages = append(messages, fmt.Sprintf("It dealt %d%% damage", attackPercent))

		if effectivenessText != "" {
			messages = append(messages, effectivenessText)
		}

		// TODO: Setup state updates so that this can be in its own separate update
		// (probably make update functions return []StateUpdate)
		ailment, ok := game.STATUS_NAME_MAP[move.Meta.Ailment.Name]
		if ok && defPokemon.Status == game.STATUS_NONE {
			ailmentCheck := rand.Intn(100)
			ailmentChance := move.Meta.AilmentChance

			// in pokeapi speak, 0 here means the chance is 100% (at least as it relates to moves like toxic and poison-powder)
			// might have to fix edge-cases here
			if ailmentChance == 0 {
				ailmentChance = 100
			}

			if ailmentCheck < ailmentChance {
				attackActionLogger().
					Debug().
					Int("ailmentCheck", accuracyCheck).
					Int("AilmentChance", ailmentChance).
					Msg("Check succeeded")

				defPokemon.Status = ailment

				// Post-Ailment initialization
				switch defPokemon.Status {
				// Set how many turns the pokemon is asleep for
				case game.STATUS_SLEEP:
					randTime := rand.Intn(2) + 1
					defPokemon.SleepCount = randTime
				case game.STATUS_TOXIC:
					defPokemon.ToxicCount = 1
				}

				attackActionLogger().Info().
					Msgf("%s was afflicted with ailment: %s:%d", defPokemon.Nickname, move.Meta.Ailment.Name, ailment)
			} else {
				attackActionLogger().
					Debug().
					Int("ailmentCheck", accuracyCheck).
					Int("AilmentChance", ailmentChance).
					Msg("Check failed")
			}
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", move.Accuracy).Msg("Check failed")
		messages = append(messages, "It missed!")
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

type SleepAction struct {
	ctx ActionCtx
}

func NewSleepAction(playerId int) *SleepAction {
	return &SleepAction{
		ctx: NewActionCtx(playerId),
	}
}

func (a SleepAction) UpdateState(state GameState) StateUpdate {
	poke := state.GetPlayer(a.ctx.PlayerId).GetActivePokemon()

	poke.SleepCount--

	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("%s is asleep", poke.Nickname)},
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

func (a ParaAction) UpdateState(state GameState) StateUpdate {
	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("Player %d's pokemon is paralyzed and cannot move", a.ctx.PlayerId)},
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

func (a BurnAction) UpdateState(state GameState) StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	damage := pokemon.MaxHp / 16
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("%s pokemon is burned", pokemon.Nickname), fmt.Sprintf("Burn did %d%% damage", damagePercent)},
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

func (a PoisonAction) UpdateState(state GameState) StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	// for future reference, this is MaxHp / 16 in gen 1
	damage := pokemon.MaxHp / 8
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("%s is poisoned", pokemon.Nickname), fmt.Sprintf("Poison did %d%% damage", damagePercent)},
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

func (a ToxicAction) UpdateState(state GameState) StateUpdate {
	player := state.GetPlayer(a.ctx.PlayerId)
	pokemon := player.GetActivePokemon()

	damage := (pokemon.MaxHp / 16) * uint(pokemon.ToxicCount)
	pokemon.Damage(damage)
	damagePercent := uint((float32(damage) / float32(pokemon.MaxHp)) * 100)

	log.Info().Int("toxicCount", pokemon.ToxicCount).Uint("damage", damage).Msg("Toxic Action")

	pokemon.ToxicCount++

	return StateUpdate{
		State:    state,
		Messages: []string{fmt.Sprintf("%s is badly poisoned", pokemon.Nickname), fmt.Sprintf("Toxic did %d%% damage", damagePercent)},
	}
}
