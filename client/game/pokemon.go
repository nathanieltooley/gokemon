package game

import (
	"fmt"
	"math"

	err "errors"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var builderLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "pokemon-builder").Logger()
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
	WEATHER_NONE = iota
	WEATHER_RAIN
	WEATHER_SUN
	WEATHER_SANDSTORM
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
	Abilities     []Ability
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

func (b BasePokemon) HasType(t *PokemonType) bool {
	return b.Type1 == t || b.Type2 == t
}

type Stat struct {
	RawValue uint
	Ev       uint
	Iv       uint
	Stage    int
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

var critStateMultipliers = map[int]float32{
	1: 1.0 / 24.0,
	2: 1.0 / 8.0,
	3: 1.0 / 2.0,
	4: 1.0,
}

var evasivenessStageMult = map[int]float32{
	-6: 9.0 / 3.0,
	-5: 8.0 / 3.0,
	-4: 7.0 / 3.0,
	-3: 6.0 / 3.0,
	-2: 5.0 / 3.0,
	-1: 4.0 / 3.0,
	0:  1,
	1:  3.0 / 4.0,
	2:  3.0 / 5.0,
	3:  3.0 / 6.0,
	4:  3.0 / 7.0,
	5:  3.0 / 8.0,
	6:  3.0 / 9.0,
}

var accuracyStageMult = map[int]float32{
	6:  9.0 / 3.0,
	5:  8.0 / 3.0,
	4:  7.0 / 3.0,
	3:  6.0 / 3.0,
	2:  5.0 / 3.0,
	1:  4.0 / 3.0,
	0:  1,
	-1: 3.0 / 4.0,
	-2: 3.0 / 5.0,
	-3: 3.0 / 6.0,
	-4: 3.0 / 7.0,
	-5: 3.0 / 8.0,
	-6: 3.0 / 9.0,
}

func StageIncrease(inc int, currentStage int, maxStage int) int {
	inc = int(math.Abs(float64(inc)))
	return int(math.Min(float64(maxStage), float64(currentStage+inc)))
}

func StageDecrease(dec int, currentStage int, minStage int) int {
	dec = int(math.Abs(float64(dec)))
	return int(math.Max(float64(minStage), float64(currentStage-dec)))
}

func (s Stat) CalcValue() int {
	return int(float32(s.RawValue) * StageMultipliers[s.Stage])
}

func (s *Stat) ChangeStat(change int) {
	if change > 0 {
		s.IncreaseStage(change)
	} else {
		s.DecreaseStage(change)
	}
}

func (s *Stat) IncreaseStage(inc int) {
	s.Stage = StageIncrease(inc, s.Stage, 6)
}

func (s *Stat) DecreaseStage(dec int) {
	s.Stage = StageDecrease(dec, s.Stage, -6)
}

func (s Stat) GetStage() int {
	return s.Stage
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
	Moves              [4]Move
	Nature             Nature
	Ability            Ability
	Item               string
	Status             int           `json:"-"`
	ConfusionCount     int           `json:"-"`
	ToxicCount         int           `json:"-"`
	SleepCount         int           `json:"-"`
	CanAttackThisTurn  bool          `json:"-"`
	SwitchedInThisTurn bool          `json:"-"`
	CritStage          int           `json:"-"`
	AccuracyStage      int           `json:"-"`
	EvasionStage       int           `json:"-"`
	InGameMoveInfo     [4]BattleMove `json:"-"`
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

func (p *Pokemon) DamagePerc(dmg float64) {
	dmgAmount := float64(p.MaxHp) * dmg
	p.Damage(uint(dmgAmount))
}

func (p *Pokemon) Heal(heal uint) {
	cappedNewHealth := uint(math.Min(float64(p.MaxHp), float64(p.Hp.Value+heal)))

	p.Hp.Value = cappedNewHealth
}

func (p *Pokemon) HealPerc(heal float64) {
	healAmount := float64(p.MaxHp) * heal
	p.Heal(uint(healAmount))
}

// Get the speed of the Pokemon, accounting for effects like paralysis
func (p *Pokemon) Speed() int {
	if p.Status == STATUS_PARA {
		return p.RawSpeed.CalcValue() / 2
	} else {
		return p.RawSpeed.CalcValue()
	}
}

func (p *Pokemon) CritChance() float32 {
	mult, ok := critStateMultipliers[p.CritStage]

	if ok {
		return mult
	} else {
		return 1.0 / 24.0
	}
}

func (p *Pokemon) ChangeEvasion(change int) {
	if change < 0 {
		p.EvasionStage = StageDecrease(change, p.EvasionStage, -6)
	} else {
		p.EvasionStage = StageIncrease(change, p.EvasionStage, 6)
	}
}

func (p Pokemon) Evasion() float32 {
	return evasivenessStageMult[p.EvasionStage]
}

func (p *Pokemon) ChangeAccuracy(change int) {
	if change < 0 {
		p.AccuracyStage = StageDecrease(change, p.AccuracyStage, -6)
	} else {
		p.AccuracyStage = StageIncrease(change, p.AccuracyStage, 6)
	}
}

func (p Pokemon) Accuracy() float32 {
	return accuracyStageMult[p.AccuracyStage]
}

// Return text that should show when a pokemon's ability is activated
func (p *Pokemon) AbilityText() string {
	return fmt.Sprintf("%s activated %s!", p.Nickname, p.Ability.Name)
}

func (p Pokemon) IsNil() bool {
	return p.Base == nil
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
		return evs, fmt.Errorf("stat total (%d) is greater than the max allowed: %d", evTotal, MAX_TOTAL_EV)
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
