package golurk

type EffectEntry struct {
	Effect      string `json:"effect"`
	ShortEffect string `json:"short_effect"`
}

type Target struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type NamedApiResource struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type StatChange struct {
	Change   int    `json:"change"`
	StatName string `json:"stat_name"`
}

// For values that are pointers, they are nullable
type MoveMeta struct {
	Ailment struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"ailment"`
	AilmentChance int `json:"ailment_chance"`
	FlinchChance  int `json:"flinch_chance"`
	StatChance    int `json:"stat_chance"`
	Category      struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"category"`

	// Null means always hits once
	MinHits *int `json:"min_hits"`
	// Null means always hits once
	MaxHits *int `json:"max_hits"`
	// Null means always one turn
	MinTurns *int `json:"min_turns"`
	// Null means always one turn
	MaxTurns *int `json:"max_turns"`

	Drain         int `json:"drain"`
	Healing       int `json:"healing"`
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
	Accuracy         int                `json:"accuracy"`
	DamageClass      string             `json:"damage_class"`
	EffectChance     int                `json:"effect_chance"`
	EffectEntry      EffectEntry        `json:"effect_entry"`
	LearnedByPokemon []NamedApiResource `json:"learned_by_pokemon"`
	Meta             *MoveMeta          `json:"meta"`
	Name             string             `json:"name"`
	Power            int                `json:"power"`
	PP               int                `json:"pp"`
	Priority         int                `json:"priority"`
	StatChanges      []StatChange       `json:"stat_changes"`
	Target           Target             `json:"target"`
	Type             string             `json:"type"`
}

type Move struct {
	Accuracy     int          `json:"accuracy"`
	DamageClass  string       `json:"damage_class"`
	EffectChance int          `json:"effect_chance"`
	EffectEntry  EffectEntry  `json:"effect_entry"`
	Meta         MoveMeta     `json:"meta"`
	Name         string       `json:"name"`
	Power        int          `json:"power"`
	PP           int          `json:"pp"`
	Priority     int          `json:"priority"`
	StatChanges  []StatChange `json:"stat_changes"`
	Target       Target       `json:"target"`
	Type         string       `json:"type"`
}

func (m Move) IsNil() bool {
	return m.Name == ""
}

type BattleMove struct {
	Info Move
	PP   int
}
