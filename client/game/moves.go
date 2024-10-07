package game

import (
	"encoding/json"
	"os"
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
	moves := m.MoveMap[pokemonName]
	movesFull := make([]*MoveFull, 0, len(moves))

	for _, moveName := range moves {
		movesFull = append(movesFull, m.GetMove(moveName))
	}

	return movesFull
}

func LoadMoves(movesPath string, movesMapPath string) (MoveRegistry, error) {
	var moveRegistry MoveRegistry
	moveDataBytes, err := os.ReadFile(movesPath)

	if err != nil {
		return moveRegistry, err
	}

	moveMapBytes, err := os.ReadFile(movesMapPath)

	if err != nil {
		return moveRegistry, err
	}

	parsedMoves := make([]MoveFull, 0, 1000)
	moveMap := make(map[string][]string)

	if err := json.Unmarshal(moveDataBytes, &parsedMoves); err != nil {
		return moveRegistry, err
	}
	if err := json.Unmarshal(moveMapBytes, &moveMap); err != nil {
		return moveRegistry, err
	}

	moveRegistry.MoveList = parsedMoves
	moveRegistry.MoveMap = moveMap

	return moveRegistry, nil
}
