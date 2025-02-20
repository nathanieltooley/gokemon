package game

type EffectEntry struct {
	Effect      string
	ShortEffect string `json:"short_effect"`
}

type Target struct {
	Id          int
	Name        string
	Description string
}

type NamedApiResource struct {
	Name string
	Url  string
}

type StatChange struct {
	Change   int
	StatName string
}

// For values that are pointers, they are nullable
type MoveMeta struct {
	Ailment struct {
		Id   int
		Name string
	}
	AilmentChance int `json:"ailment_chance"`
	FlinchChance  int `json:"flinch_chance"`
	StatChance    int `json:"stat_chance"`
	Category      struct {
		Id   int
		Name string
	}

	// Null means always hits once
	MinHits *int `json:"min_hits"`
	// Null means always hits once
	MaxHits *int `json:"max_hits"`
	// Null means always one turn
	MinTurns *int `json:"min_turns"`
	// Null means always one turn
	MaxTurns *int `json:"max_turns"`

	Drain         int
	Healing       int
	CritRateBonus int `json:"crit_rate"`
}

var STATUS_NAME_MAP = map[string]int{
	"paralysis": STATUS_PARA,
	"sleep":     STATUS_SLEEP,
	"freeze":    STATUS_FROZEN,
	"burn":      STATUS_BURN,
	"poison":    STATUS_POISON,
}

var EFFECT_NAME_MAP = map[string]int{
	"confusion": EFFECT_CONFUSION,
}

var EXPLOSIVE_MOVES = []string{
	"self-destruct",
	"explosion",
	"mind-blown",
	"misty-explosion",
}

type MoveFull struct {
	Accuracy         int
	DamageClass      string `json:"damage_class"`
	EffectChance     int
	EffectEntry      EffectEntry
	LearnedByPokemon []NamedApiResource `json:"learned_by_pokemon"`
	Meta             *MoveMeta
	Name             string
	Power            int
	PP               int
	Priority         int
	StatChanges      []StatChange `json:"stat_changes"`
	Target           Target
	Type             string
}

type Move struct {
	Accuracy     int
	DamageClass  string `json:"damage_class"`
	EffectChance int
	EffectEntry  EffectEntry
	Meta         *MoveMeta
	// What is checked for nil-ness
	Name        string
	Power       int
	PP          int
	Priority    int
	StatChanges []StatChange `json:"stat_changes"`
	Target      Target
	Type        string
}

func (m Move) IsNil() bool {
	return m.Name == ""
}

type BattleMove struct {
	Info Move
	PP   uint
}
