package golurk

import (
	"math/rand/v2"
	"strings"
)

func DefaultTeam() []Pokemon {
	defaultTeam := make([]Pokemon, 0)

	defaultMove := GlobalData.GetMove("tackle")
	defaultTeam = append(defaultTeam, NewPokeBuilder(GlobalData.GetPokemonByPokedex(1), internalRng).Build())
	defaultTeam = append(defaultTeam, NewPokeBuilder(GlobalData.GetPokemonByPokedex(2), internalRng).Build())

	defaultTeam[0].Moves[0] = *defaultMove

	return defaultTeam
}

func RandomTeam() []Pokemon {
	team := make([]Pokemon, 6)

	for i := range 6 {
		rndBasePkm := GlobalData.GetRandomPokemon()
		rndPkm := NewPokeBuilder(&rndBasePkm, rand.New(internalRng)).
			SetRandomEvs().
			SetRandomIvs().
			SetRandomLevel(80, 100).
			SetRandomNature().
			SetRandomMoves(GlobalData.GetFullMovesForPokemon(rndBasePkm.Name)).
			SetRandomAbility(GlobalData.GetPokemonAbilities(strings.ToLower(rndBasePkm.Name))).
			Build()
		team[i] = rndPkm
	}

	return team
}

func NewState(localTeam []Pokemon, opposingTeam []Pokemon, seed rand.PCG) GameState {
	// Make sure pokemon are inited correctly
	for i, p := range localTeam {
		p.CanAttackThisTurn = true

		for i, m := range p.Moves {
			if !m.IsNil() {
				p.InGameMoveInfo[i] = BattleMove{
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
				p.InGameMoveInfo[i] = BattleMove{
					Info: m,
					PP:   m.PP,
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
		RngSource:    seed,
	}
}
