package game

import (
	"fmt"
	"math"
	"math/rand/v2"

	err "errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

var builderLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-builder").Logger()
	return &logger
}

var damageLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-damage").Logger()
	return &logger
}

type PokemonType struct {
	Name          string
	Effectiveness map[string]float32
}

// The effectiveness of this type against some defense type
func (t PokemonType) AttackEffectiveness(defenseType string) float32 {
	effectiveness, ok := t.Effectiveness[defenseType]

	if !ok {
		log.Warn().Msgf("Could not find type effectiveness relationship: %s -> %s", t.Name, defenseType)
		return 1
	} else {
		return effectiveness
	}
}

type BasePokemon struct {
	PokedexNumber int16
	Name          string
	Type1         *PokemonType
	Type2         *PokemonType
	Hp            int16
	Attack        int16
	Def           int16
	SpAttack      int16
	SpDef         int16
	Speed         int16
}

func (b BasePokemon) DefenseEffectiveness(attackType *PokemonType) float32 {
	effectiveness1 := attackType.AttackEffectiveness(b.Type1.Name)

	var effectiveness2 float32 = 1
	if b.Type2 != nil {
		effectiveness2 = attackType.AttackEffectiveness(b.Type2.Name)
	}

	return effectiveness1 * effectiveness2
}

type Stat struct {
	Value int16
	Ev    uint8
	Iv    uint8
}

type Nature struct {
	name          string
	statModifiers [5]float32
}

type Pokemon struct {
	Base     *BasePokemon
	Nickname string
	Level    uint8
	Hp       Stat
	MaxHp    int16
	Attack   Stat
	Def      Stat
	SpAttack Stat
	SpDef    Stat
	Speed    Stat
	Moves    [4]*Move
	Nature   Nature
	Ability  string
	Item     string
}

func (p *Pokemon) ReCalcStats() {
	hpNumerator := ((2*p.Base.Hp + int16(p.Hp.Iv) + int16(p.Hp.Ev/4)) * int16(p.Level))
	p.Hp.Value = (hpNumerator / 100) + int16(p.Level) + 10
	p.MaxHp = p.Hp.Value

	p.Attack.Value = calcStat(p.Base.Attack, p.Level, p.Attack.Iv, p.Attack.Ev, p.Nature.statModifiers[0])
	p.Def.Value = calcStat(p.Base.Def, p.Level, p.Def.Iv, p.Def.Ev, p.Nature.statModifiers[0])
	p.SpAttack.Value = calcStat(p.Base.SpAttack, p.Level, p.SpAttack.Iv, p.SpAttack.Ev, p.Nature.statModifiers[0])
	p.SpDef.Value = calcStat(p.Base.SpDef, p.Level, p.SpDef.Iv, p.SpDef.Ev, p.Nature.statModifiers[0])
	p.Speed.Value = calcStat(p.Base.Speed, p.Level, p.Speed.Iv, p.Speed.Ev, p.Nature.statModifiers[0])
}

func (p Pokemon) GetCurrentEvTotal() int {
	return int(p.Hp.Ev) + int(p.Attack.Ev) + int(p.Def.Ev) + int(p.SpAttack.Ev) + int(p.SpDef.Ev) + int(p.Speed.Ev)
}

func (p Pokemon) Alive() bool {
	return p.Hp.Value > 0
}

type PokemonBuilder struct {
	poke Pokemon
}

func NewPokeBuilder(base *BasePokemon) *PokemonBuilder {
	poke := Pokemon{
		Base:     base,
		Nickname: base.Name,
		Level:    1,
		Hp:       Stat{0, 0, 0},
		Attack:   Stat{0, 0, 0},
		Def:      Stat{0, 0, 0},
		SpAttack: Stat{0, 0, 0},
		SpDef:    Stat{0, 0, 0},
		Speed:    Stat{0, 0, 0},
		Nature:   NATURE_HARDY,
	}

	return &PokemonBuilder{poke}
}

func (pb *PokemonBuilder) SetEvs(evs [6]uint8) *PokemonBuilder {
	pb.poke.Hp.Ev = evs[0]
	pb.poke.Attack.Ev = evs[1]
	pb.poke.Def.Ev = evs[2]
	pb.poke.SpAttack.Ev = evs[3]
	pb.poke.SpDef.Ev = evs[4]
	pb.poke.Speed.Ev = evs[5]

	builderLogger().Debug().
		Uint8("HP", evs[0]).
		Uint8("ATTACK", evs[1]).
		Uint8("DEF", evs[2]).
		Uint8("SPATTACK", evs[3]).
		Uint8("SPDEF", evs[4]).
		Uint8("SPEED", evs[5]).Msg("Setting EVs")

	return pb
}

func (pb *PokemonBuilder) SetIvs(ivs [6]uint8) *PokemonBuilder {
	pb.poke.Hp.Iv = ivs[0]
	pb.poke.Attack.Iv = ivs[1]
	pb.poke.Def.Iv = ivs[2]
	pb.poke.SpAttack.Iv = ivs[3]
	pb.poke.SpDef.Iv = ivs[4]
	pb.poke.Speed.Iv = ivs[5]

	builderLogger().Debug().
		Uint8("HP", ivs[0]).
		Uint8("ATTACK", ivs[1]).
		Uint8("DEF", ivs[2]).
		Uint8("SPATTACK", ivs[3]).
		Uint8("SPDEF", ivs[4]).
		Uint8("SPEED", ivs[5]).Msg("Setting IVs")

	return pb
}

func (pb *PokemonBuilder) SetPerfectIvs() *PokemonBuilder {
	pb.poke.Hp.Iv = MAX_IV
	pb.poke.Attack.Iv = MAX_IV
	pb.poke.Def.Iv = MAX_IV
	pb.poke.SpAttack.Iv = MAX_IV
	pb.poke.SpDef.Iv = MAX_IV
	pb.poke.Speed.Iv = MAX_IV

	builderLogger().Debug().Msg("Setting Perfect IVS")

	return pb
}

func (pb *PokemonBuilder) SetRandomIvs() *PokemonBuilder {
	var ivs [6]uint8

	for i := range ivs {
		iv := rand.UintN(MAX_IV + 1)
		ivs[i] = uint8(iv)
	}

	builderLogger().Debug().Msg("Setting Random IVs")
	pb.SetIvs(ivs)

	return pb
}

// Returns an array of EV spreads whose total == 510
// and follow the order of HP, ATTACK, DEF, SPATTACK, SPDEF, SPEED
func (pb *PokemonBuilder) SetRandomEvs() *PokemonBuilder {
	evPool := MAX_TOTAL_EV
	var evs [6]uint8

	for evPool > 0 {
		// randomly select a stat to add EVs to
		randomIndex := rand.UintN(6)
		currentEv := evs[randomIndex]

		remainingEvSpace := MAX_EV - currentEv

		if remainingEvSpace <= 0 {
			continue
		}

		// Get a random value to increase the EV by
		// ranges from 0 to (remainingEvSpace or MAX_EV) + 1
		randomEv := rand.UintN(uint(math.Max(float64(remainingEvSpace), MAX_EV)) + 1)
		evs[randomIndex] += uint8(randomEv)
		evPool -= int(randomEv)
	}

	builderLogger().Debug().Msg("Setting Random EVs")
	pb.SetEvs(evs)

	builderLogger().Debug().Msgf("EV Total: %d", pb.poke.GetCurrentEvTotal())
	return pb
}

func (pb *PokemonBuilder) SetLevel(level uint8) *PokemonBuilder {
	pb.poke.Level = level
	return pb
}

func (pb *PokemonBuilder) SetRandomLevel(low int, high int) *PokemonBuilder {
	n := high - low
	rndLevel := rand.IntN(n) + low
	pb.poke.Level = uint8(rndLevel)

	return pb
}

func (pb *PokemonBuilder) SetNature(nature Nature) *PokemonBuilder {
	pb.poke.Nature = nature
	return pb
}

func (pb *PokemonBuilder) SetRandomNature() *PokemonBuilder {
	rndNature := NATURES[rand.IntN(len(NATURES))]
	pb.poke.Nature = rndNature

	return pb
}

func (pb *PokemonBuilder) SetRandomMoves(possibleMoves []*Move) *PokemonBuilder {
	var moves [4]*Move

	if len(possibleMoves) == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available moves to randomize with!")
		return pb
	}

	for i := 0; i < 4; i++ {
		move := possibleMoves[rand.IntN(len(possibleMoves))]
		moves[i] = move
	}

	moveNames := lo.Map(moves[:], func(move *Move, _ int) string {
		return move.Name
	})

	builderLogger().Debug().Strs("Moves", moveNames)

	pb.poke.Moves = moves

	return pb
}

func (pb *PokemonBuilder) SetRandomAbility(possibleAbilities []string) *PokemonBuilder {
	if len(possibleAbilities) == 0 {
		builderLogger().Warn().Msg("This Pokemon was given no available abilities to randomize with!")
		return pb
	}

	pb.poke.Ability = possibleAbilities[rand.IntN(len(possibleAbilities))]
	return pb
}

// TODO: SetRandomItem

func (pb *PokemonBuilder) Build() Pokemon {
	pb.poke.ReCalcStats()
	builderLogger().Debug().Msg("Building pokemon")
	return pb.poke
}

// type PokemonRegistry struct {
// 	pkm []BasePokemon
// }

func calcStat(baseValue int16, level uint8, iv uint8, ev uint8, natureMod float32) int16 {
	statNumerator := (2*baseValue + int16(iv) + int16(ev/4)) * int16(level)
	statValue := float32((statNumerator/100)+5) * natureMod
	return int16(statValue)
}

func CreateEVSpread(hp uint, attack uint, def uint, spAttack uint, spDef uint, speed uint) ([6]uint8, error) {
	var evs [6]uint8
	if hp > MAX_EV {
		return evs, err.New("hp is too high")
	}
	if attack > MAX_EV {
		return evs, err.New("attack is too high")
	}
	if def > MAX_EV {
		return evs, err.New("def is too high")
	}
	if spAttack > MAX_EV {
		return evs, err.New("special attack is too high")
	}
	if spDef > MAX_EV {
		return evs, err.New("special defense is too high")
	}
	if speed > MAX_EV {
		return evs, err.New("speed is too high")
	}

	evTotal := hp + attack + def + spAttack + spDef + speed

	if evTotal > MAX_TOTAL_EV {
		return evs, err.New(fmt.Sprintf("stat total (%d) is greater than the max allowed: %d\n", evTotal, MAX_TOTAL_EV))
	}

	evs[0] = uint8(hp)
	evs[1] = uint8(attack)
	evs[2] = uint8(def)
	evs[3] = uint8(spAttack)
	evs[4] = uint8(spDef)
	evs[5] = uint8(speed)

	return evs, nil
}

func CreateIVSpread(hp uint, attack uint, def uint, spAttack uint, spDef uint, speed uint) ([6]uint8, error) {
	var ivs [6]uint8
	if hp > MAX_IV {
		return ivs, err.New("hp is too high")
	}
	if attack > MAX_IV {
		return ivs, err.New("attack is too high")
	}
	if def > MAX_IV {
		return ivs, err.New("def is too high")
	}
	if spAttack > MAX_IV {
		return ivs, err.New("special attack is too high")
	}
	if spDef > MAX_IV {
		return ivs, err.New("special defense is too high")
	}
	if speed > MAX_IV {
		return ivs, err.New("speed is too high")
	}

	ivs[0] = uint8(hp)
	ivs[1] = uint8(attack)
	ivs[2] = uint8(def)
	ivs[3] = uint8(spAttack)
	ivs[4] = uint8(spDef)
	ivs[5] = uint8(speed)

	return ivs, nil
}

func Damage(attacker Pokemon, defendent Pokemon, move *Move) uint {
	attackerLevel := attacker.Level // TODO: Add exception for Beat Up
	var a, d int16                  // TODO: Add exception for Beat Up

	// Determine damage type
	if move.DamageClass == DAMAGETYPE_PHYSICAL {
		a = attacker.Base.Attack
	} else if move.DamageClass == DAMAGETYPE_SPECIAL {
		a = attacker.Base.SpAttack
	}

	if move.DamageClass == DAMAGETYPE_PHYSICAL {
		d = defendent.Base.Def
	} else if move.DamageClass == DAMAGETYPE_SPECIAL {
		d = defendent.Base.SpDef
	}

	power := move.Power

	if power == 0 {
		return 0
	}

	damageLogger().Debug().Msgf("Type 1: %#v", defendent.Base.Type1)
	damageLogger().Debug().Msgf("Type 2: %#v", defendent.Base.Type2)

	attackType := GetAttackTypeMapping(move.Type)

	type1Effectiveness := attackType.AttackEffectiveness(defendent.Base.Type1.Name)

	var type2Effectiveness float32 = 1

	if defendent.Base.Type2 != nil {
		type2Effectiveness = attackType.AttackEffectiveness(defendent.Base.Type2.Name)
	}

	if type1Effectiveness == 0 || type2Effectiveness == 0 {
		return 0
	}

	// Calculate the part of the damage function in brackets
	damageInner := ((((float32(2*attackerLevel) / 5) + 2) * float32(power) * (float32(a) / float32(d))) / 50) + 2
	randomSpread := float32(rand.UintN(15)+85) / 100
	var stab float32 = 1

	if move.Type == attacker.Base.Type1.Name || (attacker.Base.Type2 != nil && move.Type == attacker.Base.Type2.Name) {
		stab = 1.5
	}

	// TODO: Add crits, exceptions for Battle Armor and Shell Armor
	// TODO: Add burn check for 1/2 damage which needs its own expection for Guts and Facade

	// TODO: Maybe add weather
	// TODO: Maybe check for parental bond, glaive rush, targets in DBs, ZMoves

	// TODO: There are about 20 different moves, abilities, and items that have some sort of
	// random effect to the damage calcs. Maybe implement the most impactful ones?

	// This seems to mostly work, however there are issues when it comes to rounding
	// and it seems that the lowest possible value in a damage range may not be able
	// to show up as often because rounding is a bit different
	// TODO: maybe make a custom rounding function that rounds DOWN at .5
	damage := uint(math.Round(float64(damageInner * randomSpread * type1Effectiveness * type2Effectiveness * stab)))

	damageLogger().Debug().
		Int("power", power).
		Uint8("attackerLevel", attackerLevel).
		Float32("damageInner", damageInner).
		Float32("randomSpread", randomSpread).
		Float32("STAB", stab).
		Float32("Net Type Effectiveness", type1Effectiveness*type2Effectiveness).
		Uint("damage", damage).
		Msg("")

	return damage
}
