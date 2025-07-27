// Package core (thereafter refered to as game.core to avoid confusion with state.core) contains all foundational data,
// types and functions for dealing with game data. game.core CANNOT have dependencies with other packages in this project (some expections may apply).
// This is a hard rule due to cyclic dependencies, the reason this package was separated in the first place.
package core

import (
	"fmt"
	"math"

	err "errors"

	"github.com/rs/zerolog/log"
)

type PokemonType struct {
	Name          string
	Effectiveness map[string]float64
}

// AttackEffectiveness gives the type effectiveness of an attack of this type compared to a given defense type.
// For instance, if self's type is Grass, this function gets the type effectiveness of a Grass attack against a given type (i.e Water would 2X effective, Fire would 1/2 effective)
func (t PokemonType) AttackEffectiveness(defenseType PokemonType) float64 {
	effectiveness, ok := t.Effectiveness[defenseType.Name]

	if !ok {
		log.Warn().Msgf("Could not find type effectiveness relationship: %s -> %s", t.Name, defenseType.Name)
		return 1
	} else {
		return effectiveness
	}
}

// BasePokemon is the base stats, possible abilities, and type information of a Pokemon, as if it were a PokeDex entry.
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

// HasType returns whether a BasePokemon is either of these types
func (b BasePokemon) HasType(t *PokemonType) bool {
	return b.Type1 == t || b.Type2 == t
}

// Stat represents one of a Pokemon's stats, keeping track of the raw values and also stage modifers.
type Stat struct {
	RawValue uint
	Ev       uint
	Iv       uint
	Stage    int `json:"-"`
}

// HpStat is a special version of Stat that does not contain stage info as HP cannot be increased in stages like other stats.
type HpStat struct {
	Value uint
	Ev    uint
	Iv    uint
}

// CalcValue gets the final, actual value of a Pokemon's stat after being modified by it's stage.
func (s Stat) CalcValue() int {
	return int(float32(s.RawValue) * StageMultipliers[s.Stage])
}

// ChangeStat is a automatic version of Stat.IncreaseStage and Stat.DecreaseStage, calling StageIncrease when change is positive and StageDecrease when change is negative.
func (s *Stat) ChangeStat(change int) {
	if change > 0 {
		s.IncreaseStage(change)
	} else {
		s.DecreaseStage(change)
	}
}

// IncreaseStage calls StageIncrease with a max stage of 6
func (s *Stat) IncreaseStage(inc int) {
	s.Stage = stageIncrease(inc, s.Stage, 6)
}

// DecreaseStage calls StageDecrease with a min stage of -6
func (s *Stat) DecreaseStage(dec int) {
	s.Stage = stageDecrease(dec, s.Stage, -6)
}

func (s Stat) GetStage() int {
	return s.Stage
}

type Nature struct {
	Name          string
	StatModifiers [5]float32
}

// Pokemon is a mixture of an edited Pokemon on a team, and a version of that Pokemon in a battle.
// Certain values that are only relevant to battles (like stat stages, counters for sleep and toxic, or PP [lol] for a move) are not saved as team data.
type Pokemon struct {
	Base                 *BasePokemon
	Nickname             string
	Level                uint
	Hp                   HpStat
	MaxHp                uint
	Attack               Stat
	Def                  Stat
	SpAttack             Stat
	SpDef                Stat
	RawSpeed             Stat
	Moves                [4]Move
	Nature               Nature
	Ability              Ability
	Item                 string
	BattleType           *PokemonType  `json:"-"`
	Status               int           `json:"-"`
	ConfusionCount       int           `json:"-"`
	ToxicCount           int           `json:"-"`
	SleepCount           int           `json:"-"`
	CanAttackThisTurn    bool          `json:"-"`
	SwitchedInThisTurn   bool          `json:"-"`
	CritStage            int           `json:"-"`
	AccuracyStage        int           `json:"-"`
	EvasionStage         int           `json:"-"`
	InGameMoveInfo       [4]BattleMove `json:"-"`
	FlashFire            bool          `json:"-"`
	TruantShouldActivate bool          `json:"-"`
}

func (p Pokemon) HasType(pokemonType *PokemonType) bool {
	// end early if there is a battle type
	if p.BattleType != nil {
		return p.BattleType.Name == pokemonType.Name
	}

	if p.Base.Type1 != nil && p.Base.Type1.Name == pokemonType.Name {
		return true
	}

	if p.Base.Type2 != nil && p.Base.Type2.Name == pokemonType.Name {
		return true
	}

	return false
}

func (p *Pokemon) ReCalcStats() {
	if p.Base.Name == "shedinja" {
		p.Hp.Value = 1
		p.MaxHp = 1
	} else {
		hpNumerator := (2*p.Base.Hp + p.Hp.Iv + (p.Hp.Ev / 4)) * (p.Level)
		p.Hp.Value = (hpNumerator / 100) + p.Level + 10
		p.MaxHp = p.Hp.Value
	}

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
	dmgAmount := math.Ceil(float64(p.MaxHp) * dmg)
	p.Damage(uint(dmgAmount))
}

func (p *Pokemon) Heal(heal uint) {
	cappedNewHealth := uint(math.Min(float64(p.MaxHp), float64(p.Hp.Value+heal)))

	p.Hp.Value = cappedNewHealth
}

func (p *Pokemon) HealPerc(heal float64) {
	healAmount := math.Ceil(float64(p.MaxHp) * heal)
	p.Heal(uint(healAmount))
}

func (p *Pokemon) Speed(weather int) int {
	calcedSpeed := p.RawSpeed.CalcValue()

	if p.Status == STATUS_PARA {
		calcedSpeed = calcedSpeed / 2
	}

	if p.Ability.Name == "swift-swim" && weather == WEATHER_RAIN {
		calcedSpeed = calcedSpeed * 2
	}

	if p.Ability.Name == "chlorophyll" && weather == WEATHER_SUN {
		calcedSpeed = calcedSpeed * 2
	}

	return calcedSpeed
}

// CritChance gets the chance for a Pokemon's move to crit based on the Pokemon's current crit change stage
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
		p.EvasionStage = stageDecrease(change, p.EvasionStage, -6)
	} else {
		p.EvasionStage = stageIncrease(change, p.EvasionStage, 6)
	}
}

func (p Pokemon) Evasion() float32 {
	return evasivenessStageMult[p.EvasionStage]
}

func (p *Pokemon) ChangeAccuracy(change int) {
	if change < 0 {
		p.AccuracyStage = stageDecrease(change, p.AccuracyStage, -6)
	} else {
		p.AccuracyStage = stageIncrease(change, p.AccuracyStage, 6)
	}
}

func (p Pokemon) Accuracy() float32 {
	return accuracyStageMult[p.AccuracyStage]
}

// AbilityText returns text that should show when a pokemon's ability is activated
func (p *Pokemon) AbilityText() string {
	return fmt.Sprintf("%s activated %s!", p.Nickname, p.Ability.Name)
}

func (p Pokemon) IsNil() bool {
	return p.Base == nil
}

// DefenseEffectiveness gets the type effectiveness of an attackType against this BasePokemon's one or two types
func (p Pokemon) DefenseEffectiveness(attackType *PokemonType) float64 {
	if p.BattleType != nil {
		return attackType.AttackEffectiveness(*p.BattleType)
	} else {
		effectiveness1 := attackType.AttackEffectiveness(*p.Base.Type1)

		var effectiveness2 float64 = 1
		if p.Base.Type2 != nil {
			effectiveness2 = attackType.AttackEffectiveness(*p.Base.Type2)
		}

		return effectiveness1 * effectiveness2
	}
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

func calcStat(baseValue uint, level uint, iv uint, ev uint, natureMod float32) uint {
	statNumerator := (2*baseValue + iv + (ev / 4)) * (level)
	statValue := (float32(statNumerator)/100 + 5) * natureMod
	log.Debug().Float32("stat", statValue).Msg("")
	return uint(statValue)
}

// stageIncrease increases a stat's stage, keeping in mind the max value for the stat.
// Generalized for a stat with any given max value as crit chance stages only go up to 4.
func stageIncrease(inc int, currentStage int, maxStage int) int {
	inc = int(math.Abs(float64(inc)))
	return int(math.Min(float64(maxStage), float64(currentStage+inc)))
}

// stageDecrease decreases a stat's stage, keeping in mind the min value for the stat.
// Generalized for a stat with any given min value.
func stageDecrease(dec int, currentStage int, minStage int) int {
	dec = int(math.Abs(float64(dec)))
	return int(math.Max(float64(minStage), float64(currentStage-dec)))
}
