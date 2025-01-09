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

const (
	STATUS_NONE = iota
	STATUS_BURN
	// idea for both para and sleep:
	// when a move gets sent as an action
	// theres a chance the move action turns into
	// a para or sleep action (functionally same as skip but with different messages)
	STATUS_PARA
	STATUS_SLEEP
	STATUS_FROZEN
	// will have to check at the end of a turn for damage
	STATUS_POISON
	STATUS_TOXIC
)

const (
	EFFECT_CONFUSION = iota
)

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
	PokedexNumber uint
	Name          string
	Type1         *PokemonType
	Type2         *PokemonType
	Hp            uint
	Attack        uint
	Def           uint
	SpAttack      uint
	SpDef         uint
	Speed         uint
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
	RawValue uint
	Ev       uint
	Iv       uint
	stage    int
}

type HpStat struct {
	Value uint
	Ev    uint
	Iv    uint
}

var StageMultipliers = map[int]float32{
	-6: 2.0 / 8.0,
	-5: 2.0 / 7.0,
	-4: 2.0 / 6.0,
	-3: 2.0 / 5.0,
	-2: 2.0 / 4.0,
	-1: 2.0 / 3.0,
	0:  1,
	1:  3.0 / 2.0,
	2:  4.0 / 2.0,
	3:  5.0 / 2.0,
	4:  6.0 / 2.0,
	5:  7.0 / 2.0,
	6:  8.0 / 2.0,
}

func (s Stat) CalcValue() int {
	return int(float32(s.RawValue) * StageMultipliers[s.stage])
}

func (s *Stat) ChangeStat(change int) {
	if change > 0 {
		s.IncreaseStage(change)
	} else {
		s.DecreaseStage(change)
	}
}

func (s *Stat) IncreaseStage(inc int) {
	inc = int(math.Abs(float64(inc)))
	s.stage = int(math.Min(6, float64(s.stage+inc)))
}

func (s *Stat) DecreaseStage(dec int) {
	dec = int(math.Abs(float64(dec)))
	s.stage = int(math.Max(-6, float64(s.stage-dec)))
}

func (s Stat) GetStage() int {
	return s.stage
}

type Nature struct {
	Name          string
	StatModifiers [5]float32
}

type Pokemon struct {
	Base               *BasePokemon
	Nickname           string
	Level              uint
	Hp                 HpStat
	MaxHp              uint
	Attack             Stat
	Def                Stat
	SpAttack           Stat
	SpDef              Stat
	RawSpeed           Stat
	Moves              [4]*Move
	Nature             Nature
	Ability            string
	Item               string
	Status             int  `json:"-"`
	ConfusionCount     int  `json:"-"`
	ToxicCount         int  `json:"-"`
	SleepCount         int  `json:"-"`
	CanAttackThisTurn  bool `json:"-"`
	SwitchedInThisTurn bool `json:"-"`
	InGameMoveInfo     [4]BattleMove
}

func (p *Pokemon) ReCalcStats() {
	hpNumerator := (2*p.Base.Hp + p.Hp.Iv + (p.Hp.Ev / 4)) * (p.Level)
	p.Hp.Value = (hpNumerator / 100) + p.Level + 10
	p.MaxHp = p.Hp.Value

	p.Attack.RawValue = calcStat(p.Base.Attack, p.Level, p.Attack.Iv, p.Attack.Ev, p.Nature.StatModifiers[0])
	p.Def.RawValue = calcStat(p.Base.Def, p.Level, p.Def.Iv, p.Def.Ev, p.Nature.StatModifiers[0])
	p.SpAttack.RawValue = calcStat(p.Base.SpAttack, p.Level, p.SpAttack.Iv, p.SpAttack.Ev, p.Nature.StatModifiers[0])
	p.SpDef.RawValue = calcStat(p.Base.SpDef, p.Level, p.SpDef.Iv, p.SpDef.Ev, p.Nature.StatModifiers[0])
	p.RawSpeed.RawValue = calcStat(p.Base.Speed, p.Level, p.RawSpeed.Iv, p.RawSpeed.Ev, p.Nature.StatModifiers[0])
}

func (p Pokemon) GetCurrentEvTotal() int {
	return int(p.Hp.Ev) + int(p.Attack.Ev) + int(p.Def.Ev) + int(p.SpAttack.Ev) + int(p.SpDef.Ev) + int(p.RawSpeed.Ev)
}

func (p Pokemon) Alive() bool {
	return p.Hp.Value > 0
}

func (p *Pokemon) Damage(dmg uint) {
	cappedNewHealth := uint(math.Max(0, float64(int(p.Hp.Value)-int(dmg))))

	log.Debug().Uint("dmg", dmg).Uint("oldHealth", p.Hp.Value).Uint("cappedNewHealth", cappedNewHealth).Msg("pkm damage")

	p.Hp.Value = cappedNewHealth
}

func (p *Pokemon) Heal(heal uint) {
	cappedNewHealth := uint(math.Min(float64(p.MaxHp), float64(p.Hp.Value+heal)))

	p.Hp.Value = cappedNewHealth
}

// Get the speed of the Pokemon, accounting for effects like paralysis
func (p *Pokemon) Speed() int {
	if p.Status == STATUS_PARA {
		return p.RawSpeed.CalcValue() / 2
	} else {
		return p.RawSpeed.CalcValue()
	}
}

type PokemonBuilder struct {
	poke Pokemon
}

func NewPokeBuilder(base *BasePokemon) *PokemonBuilder {
	poke := Pokemon{
		Base:     base,
		Nickname: base.Name,
		Level:    1,
		Hp:       HpStat{0, 0, 0},
		Attack:   Stat{0, 0, 0, 0},
		Def:      Stat{0, 0, 0, 0},
		SpAttack: Stat{0, 0, 0, 0},
		SpDef:    Stat{0, 0, 0, 0},
		RawSpeed: Stat{0, 0, 0, 0},
		Nature:   NATURE_HARDY,
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
		iv := rand.UintN(MAX_IV + 1)
		ivs[i] = iv
	}

	builderLogger().Debug().Msg("Setting Random IVs")
	pb.SetIvs(ivs)

	return pb
}

// Returns an array of EV spreads whose total == 510
// and follow the order of HP, ATTACK, DEF, SPATTACK, SPDEF, SPEED
func (pb *PokemonBuilder) SetRandomEvs() *PokemonBuilder {
	evPool := MAX_TOTAL_EV
	var evs [6]uint

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
	var n uint = uint(high - low)
	rndLevel := rand.UintN(n) + uint(low)
	pb.poke.Level = rndLevel

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

func calcStat(baseValue uint, level uint, iv uint, ev uint, natureMod float32) uint {
	statNumerator := (2*baseValue + iv + (ev / 4)) * (level)
	statValue := (float32(statNumerator)/100 + 5) * natureMod
	log.Debug().Float32("stat", statValue).Msg("")
	return uint(statValue)
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

func Damage(attacker Pokemon, defendent Pokemon, move *Move, crit bool) uint {
	attackerLevel := attacker.Level // TODO: Add exception for Beat Up
	var baseA, baseD uint
	var a, d uint // TODO: Add exception for Beat Up
	var aBoost, dBoost int

	// Determine damage type
	if move.DamageClass == DAMAGETYPE_PHYSICAL {
		baseA = attacker.Attack.RawValue
		a = uint(attacker.Attack.CalcValue())
		aBoost = attacker.Attack.stage
	} else if move.DamageClass == DAMAGETYPE_SPECIAL {
		baseA = attacker.SpAttack.RawValue
		a = uint(attacker.SpAttack.CalcValue())
		aBoost = attacker.SpAttack.stage
	}

	if move.DamageClass == DAMAGETYPE_PHYSICAL {
		baseD = defendent.Def.RawValue
		d = uint(defendent.Def.CalcValue())
		dBoost = defendent.Def.stage
	} else if move.DamageClass == DAMAGETYPE_SPECIAL {
		baseD = defendent.SpDef.RawValue
		d = uint(defendent.SpDef.CalcValue())
		dBoost = defendent.SpDef.stage
	}

	power := move.Power

	if power == 0 {
		return 0
	}

	damageLogger().Debug().Msgf("Type 1: %#v", defendent.Base.Type1)
	damageLogger().Debug().Msgf("Type 2: %#v", defendent.Base.Type2)

	attackType := GetAttackTypeMapping(move.Type)

	var type1Effectiveness float32 = 1
	var type2Effectiveness float32 = 1

	if attackType != nil {
		type1Effectiveness = attackType.AttackEffectiveness(defendent.Base.Type1.Name)

		if defendent.Base.Type2 != nil {
			type2Effectiveness = attackType.AttackEffectiveness(defendent.Base.Type2.Name)
		}
	}

	if type1Effectiveness == 0 || type2Effectiveness == 0 {
		return 0
	}

	var critBoost float32 = 1
	if crit && defendent.Ability != "battle-armor" && defendent.Ability != "shell-armor" {
		critBoost = 1.5
		a = baseA
		d = baseD
	} else if crit && (defendent.Ability == "battle-armor" || defendent.Ability == "shell-armor") {
		damageLogger().Info().Msgf("Defending pokemon blocked crits with %s", defendent.Ability)
	}

	// Calculate the part of the damage function in brackets
	damageInner := ((((float32(2*attackerLevel) / 5) + 2) * float32(power) * (float32(a) / float32(d))) / 50) + 2
	randomSpread := float32(rand.UintN(15)+85) / 100
	var stab float32 = 1

	if move.Type == attacker.Base.Type1.Name || (attacker.Base.Type2 != nil && move.Type == attacker.Base.Type2.Name) {
		stab = 1.5
	}

	var burn float32 = 1
	// TODO: Add exception for Guts and Facade
	if attacker.Status == STATUS_BURN && move.DamageClass == DAMAGETYPE_PHYSICAL {
		burn = 0.5
		damageLogger().Info().Float32("burn", burn).Msg("Attacker is burned and is using a physical move")
	}

	// TODO: Maybe add weather
	// TODO: Maybe check for parental bond, glaive rush, targets in DBs, ZMoves

	// TODO: There are about 20 different moves, abilities, and items that have some sort of
	// random effect to the damage calcs. Maybe implement the most impactful ones?

	// This seems to mostly work, however there are issues when it comes to rounding
	// and it seems that the lowest possible value in a damage range may not be able
	// to show up as often because rounding is a bit different
	// TODO: maybe make a custom rounding function that rounds DOWN at .5
	damage := uint(math.Round(float64(damageInner * randomSpread * type1Effectiveness * type2Effectiveness * stab * burn * critBoost)))

	damageLogger().Debug().
		Int("power", power).
		Uint("attackerLevel", attackerLevel).
		Uint("attackValue", a).
		Int("attackChange", aBoost).
		Uint("defValue", d).
		Int("defenseChange", dBoost).
		Str("attackType", move.Type).
		Float32("damageInner", damageInner).
		Float32("randomSpread", randomSpread).
		Float32("STAB", stab).
		Float32("Net Type Effectiveness", type1Effectiveness*type2Effectiveness).
		Float32("crit", critBoost).
		Uint("damage", damage).
		Msg("damage calc")

	return damage
}
