package golurk

import (
	"math"
	"math/rand/v2"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var builderLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-builder").Logger()
	return &logger
}

type PokemonBuilder struct {
	poke Pokemon
	rng  rand.Rand
}

func NewPokeBuilder(base *BasePokemon, rng *rand.Rand) *PokemonBuilder {
	poke := Pokemon{
		Base:     base,
		Nickname: base.Name,
		Level:    1,
		Hp:       HpStat{Value: 0, Ev: 0, Iv: 0},
		Attack:   Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		Def:      Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		SpAttack: Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		SpDef:    Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		RawSpeed: Stat{RawValue: 0, Ev: 0, Iv: 0, Stage: 0},
		Nature:   NATURE_HARDY,
	}

	return &PokemonBuilder{poke, *rng}
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
	pb.poke.Hp.Iv = MAX_IV
	pb.poke.Attack.Iv = MAX_IV
	pb.poke.Def.Iv = MAX_IV
	pb.poke.SpAttack.Iv = MAX_IV
	pb.poke.SpDef.Iv = MAX_IV
	pb.poke.RawSpeed.Iv = MAX_IV

	builderLogger().Debug().Msg("Setting Perfect IVS")

	return pb
}

func (pb *PokemonBuilder) SetRandomIvs() *PokemonBuilder {
	var ivs [6]uint

	for i := range ivs {
		iv := pb.rng.UintN(MAX_IV + 1)
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
	evPool := MAX_TOTAL_EV
	var evs [6]uint

	for evPool > 0 {
		// randomly select a stat to add EVs to
		randomIndex := pb.rng.UintN(6)
		currentEv := evs[randomIndex]

		remainingEvSpace := MAX_EV - currentEv

		if remainingEvSpace <= 0 {
			continue
		}

		// Get a random value to increase the EV by
		// ranges from 0 to (remainingEvSpace or MAX_EV) + 1
		randomEv := pb.rng.UintN(uint(math.Max(float64(remainingEvSpace), MAX_EV)) + 1)
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
	rndLevel := pb.rng.UintN(n) + uint(low)
	pb.poke.Level = rndLevel

	return pb
}

func (pb *PokemonBuilder) SetNature(nature Nature) *PokemonBuilder {
	pb.poke.Nature = nature
	return pb
}

func (pb *PokemonBuilder) SetRandomNature() *PokemonBuilder {
	rndNature := NATURES[pb.rng.IntN(len(NATURES))]
	pb.poke.Nature = rndNature

	return pb
}

func (pb *PokemonBuilder) SetRandomMoves(possibleMoves []Move) *PokemonBuilder {
	var moves [4]Move

	if len(possibleMoves) == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available moves to randomize with!")
		return pb
	}

	for i := range 4 {
		move := possibleMoves[pb.rng.IntN(len(possibleMoves))]
		moves[i] = move
	}

	moveNames := lo.Map(moves[:], func(move Move, _ int) string {
		return move.Name
	})

	builderLogger().Debug().Strs("Moves", moveNames)

	pb.poke.Moves = moves

	return pb
}

func (pb *PokemonBuilder) SetRandomAbility(possibleAbilities []Ability) *PokemonBuilder {
	abilityCount := len(possibleAbilities)
	if abilityCount == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available abilities to randomize with!")
		return pb
	}

	hiddenAbility, found := lo.Find(possibleAbilities, func(a Ability) bool {
		return a.IsHidden
	})
	normalAbilities := lo.Filter(possibleAbilities, func(a Ability, _ int) bool {
		return !a.IsHidden
	})

	choseHidden := pb.rng.Float64()

	// 1% chance to get a hidden ability randomly
	if found && choseHidden < 0.01 {
		pb.poke.Ability = hiddenAbility
	} else {
		pb.poke.Ability = normalAbilities[pb.rng.IntN(len(normalAbilities))]
	}

	return pb
}

// TODO: SetRandomItem

func (pb *PokemonBuilder) Build() Pokemon {
	pb.poke.ReCalcStats()
	builderLogger().Debug().Msg("Building pokemon")
	return pb.poke
}
