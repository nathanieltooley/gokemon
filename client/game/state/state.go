package state

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/global"
)

func DefaultTeam() []core.Pokemon {
	defaultTeam := make([]core.Pokemon, 0)

	defaultMove := global.MOVES.GetMove("tackle")
	defaultTeam = append(defaultTeam, game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(1)).Build())
	defaultTeam = append(defaultTeam, game.NewPokeBuilder(global.POKEMON.GetPokemonByPokedex(2)).Build())

	defaultTeam[0].Moves[0] = *defaultMove

	return defaultTeam
}

func RandomTeam() []core.Pokemon {
	team := make([]core.Pokemon, 6)

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

func NewState(localTeam []core.Pokemon, opposingTeam []core.Pokemon, seed rand.PCG) stateCore.GameState {
	// Make sure pokemon are inited correctly
	for i, p := range localTeam {
		p.CanAttackThisTurn = true

		for i, m := range p.Moves {
			if !m.IsNil() {
				p.InGameMoveInfo[i] = core.BattleMove{
					Info: m,
					PP:   m.PP,
				}
			}
		}

		localTeam[i] = p
	}

	for i, p := range opposingTeam {
		p.CanAttackThisTurn = true

		for i, m := range p.Moves {
			if !m.IsNil() {
				p.InGameMoveInfo[i] = core.BattleMove{
					Info: m,
					PP:   m.PP,
				}
			}
		}

		opposingTeam[i] = p
	}

	localPlayer := stateCore.Player{
		Name: "Local",
		Team: localTeam,
	}
	opposingPlayer := stateCore.Player{
		Name: "Opponent",
		Team: opposingTeam,
	}

	return stateCore.GameState{
		HostPlayer:   localPlayer,
		ClientPlayer: opposingPlayer,
		Turn:         0,
		RngSource:    seed,
	}
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
