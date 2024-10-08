package game

import (
	"strings"
)

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
	Change int
	Stat   struct {
		Name string
		Url  string
	}
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

type MoveRegistry struct {
	// TODO: Maybe make this a map
	MoveList []MoveFull
	MoveMap  map[string][]string
}

func (m *MoveRegistry) GetMove(name string) *MoveFull {
	for _, move := range m.MoveList {
		if move.Name == name {
			return &move
		}
	}

	return nil
}

func (m *MoveRegistry) GetFullMovesForPokemon(pokemonName string) []*MoveFull {
	pokemonLowerName := strings.ToLower(pokemonName)
	moves := m.MoveMap[pokemonLowerName]
	movesFull := make([]*MoveFull, 0, len(moves))

	for _, moveName := range moves {
		movesFull = append(movesFull, m.GetMove(moveName))
	}

	return movesFull
}
