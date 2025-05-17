package game

import (
	"math"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var builderLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-builder").Logger()
	return &logger
}

type PokemonBuilder struct {
	poke core.Pokemon
}

func NewPokeBuilder(base *core.BasePokemon) *PokemonBuilder {
	poke := core.Pokemon{
		Base:     base,
		Nickname: base.Name,
		Level:    1,
		Hp:       core.HpStat{Value: 0, Ev: 0, Iv: 0},
		Attack:   core.Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		Def:      core.Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		SpAttack: core.Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		SpDef:    core.Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		RawSpeed: core.Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		Nature:   core.NATURE_HARDY,
	}

	return &PokemonBuilder{poke}
}

func (pb *PokemonBuilder) SetEvs(evs [6]uint) *PokemonBuilder {
	pb.poke.Hp.Ev = evs[0]
	pb.poke.Attack.Ev = evs[1]
	pb.poke.Def.Ev = evs[2]
	pb.poke.SpAttack.Ev = evs[3]
	pb.poke.SpDef.Ev = evs[4]
	pb.poke.RawSpeed.Ev = evs[5]

	builderLogger().Debug().
		Uint("HP", evs[0]).
		Uint("ATTACK", evs[1]).
		Uint("DEF", evs[2]).
		Uint("SPATTACK", evs[3]).
		Uint("SPDEF", evs[4]).
		Uint("SPEED", evs[5]).Msg("Setting EVs")

	return pb
}

func (pb *PokemonBuilder) SetIvs(ivs [6]uint) *PokemonBuilder {
	pb.poke.Hp.Iv = ivs[0]
	pb.poke.Attack.Iv = ivs[1]
	pb.poke.Def.Iv = ivs[2]
	pb.poke.SpAttack.Iv = ivs[3]
	pb.poke.SpDef.Iv = ivs[4]
	pb.poke.RawSpeed.Iv = ivs[5]

	builderLogger().Debug().
		Uint("HP", ivs[0]).
		Uint("ATTACK", ivs[1]).
		Uint("DEF", ivs[2]).
		Uint("SPATTACK", ivs[3]).
		Uint("SPDEF", ivs[4]).
		Uint("SPEED", ivs[5]).Msg("Setting IVs")

	return pb
}

func (pb *PokemonBuilder) SetPerfectIvs() *PokemonBuilder {
	pb.poke.Hp.Iv = core.MAX_IV
	pb.poke.Attack.Iv = core.MAX_IV
	pb.poke.Def.Iv = core.MAX_IV
	pb.poke.SpAttack.Iv = core.MAX_IV
	pb.poke.SpDef.Iv = core.MAX_IV
	pb.poke.RawSpeed.Iv = core.MAX_IV

	builderLogger().Debug().Msg("Setting Perfect IVS")

	return pb
}

func (pb *PokemonBuilder) SetRandomIvs() *PokemonBuilder {
	var ivs [6]uint

	for i := range ivs {
		iv := global.GokeRand.UintN(core.MAX_IV + 1)
		ivs[i] = iv
	}

	builderLogger().Debug().Msg("Setting Random IVs")
	pb.SetIvs(ivs)

	return pb
}

// Returns an array of EV spreads whose total == 510
// and follow the order of HP, ATTACK, DEF, SPATTACK, SPDEF, SPEED
// TODO: pretty sure this doesn't work
func (pb *PokemonBuilder) SetRandomEvs() *PokemonBuilder {
	evPool := core.MAX_TOTAL_EV
	var evs [6]uint

	for evPool > 0 {
		// randomly select a stat to add EVs to
		randomIndex := global.GokeRand.UintN(6)
		currentEv := evs[randomIndex]

		remainingEvSpace := core.MAX_EV - currentEv

		if remainingEvSpace <= 0 {
			continue
		}

		// Get a random value to increase the EV by
		// ranges from 0 to (remainingEvSpace or MAX_EV) + 1
		randomEv := global.GokeRand.UintN(uint(math.Max(float64(remainingEvSpace), core.MAX_EV)) + 1)
		evs[randomIndex] += randomEv
		evPool -= int(randomEv)
	}

	builderLogger().Debug().Msg("Setting Random EVs")
	pb.SetEvs(evs)

	builderLogger().Debug().Msgf("EV Total: %d", pb.poke.GetCurrentEvTotal())
	return pb
}

func (pb *PokemonBuilder) SetLevel(level uint) *PokemonBuilder {
	pb.poke.Level = level
	return pb
}

func (pb *PokemonBuilder) SetRandomLevel(low int, high int) *PokemonBuilder {
	n := uint(high - low)
	rndLevel := global.GokeRand.UintN(n) + uint(low)
	pb.poke.Level = rndLevel

	return pb
}

func (pb *PokemonBuilder) SetNature(nature core.Nature) *PokemonBuilder {
	pb.poke.Nature = nature
	return pb
}

func (pb *PokemonBuilder) SetRandomNature() *PokemonBuilder {
	rndNature := core.NATURES[global.GokeRand.IntN(len(core.NATURES))]
	pb.poke.Nature = rndNature

	return pb
}

// NOTE: takes in pointers rather than values even though pokemon struct no longer holds pointers (issues with gob)
// mainly so i have to change less stuff
func (pb *PokemonBuilder) SetRandomMoves(possibleMoves []*core.Move) *PokemonBuilder {
	var moves [4]core.Move

	if len(possibleMoves) == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available moves to randomize with!")
		return pb
	}

	for i := range 4 {
		move := possibleMoves[global.GokeRand.IntN(len(possibleMoves))]
		moves[i] = *move
	}

	moveNames := lo.Map(moves[:], func(move core.Move, _ int) string {
		return move.Name
	})

	builderLogger().Debug().Strs("Moves", moveNames)

	pb.poke.Moves = moves

	return pb
}

func (pb *PokemonBuilder) SetRandomAbility(possibleAbilities []core.Ability) *PokemonBuilder {
	abilityCount := len(possibleAbilities)
	if abilityCount == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available abilities to randomize with!")
		return pb
	}

	hiddenAbility, found := lo.Find(possibleAbilities, func(a core.Ability) bool {
		return a.IsHidden
	})
	normalAbilities := lo.Filter(possibleAbilities, func(a core.Ability, _ int) bool {
		return !a.IsHidden
	})

	choseHidden := global.GokeRand.Float64()

	// 1% chance to get a hidden ability randomly
	if found && choseHidden < 0.01 {
		pb.poke.Ability = hiddenAbility
	} else {
		pb.poke.Ability = normalAbilities[global.GokeRand.IntN(len(normalAbilities))]
	}

	return pb
}

// TODO: SetRandomItem

func (pb *PokemonBuilder) Build() core.Pokemon {
	pb.poke.ReCalcStats()
	builderLogger().Debug().Msg("Building pokemon")
	return pb.poke
}
