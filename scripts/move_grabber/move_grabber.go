package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/scripts"
	"github.com/samber/lo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type MoveMetaPre struct {
	Ailment       game.NamedApiResource
	AilmentChance int `json:"ailment_chance"`
	FlinchChance  int `json:"flinch_chance"`
	StatChance    int `json:"stat_chance"`
	Category      game.NamedApiResource

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

// Follows all important NamedApiResource values and replaces that type with their actual value
func (m *MoveMetaPre) ToFullMeta() (*game.MoveMeta, error) {
	ailmentJson, err := scripts.FollowNamedResource[struct {
		Id   int
		Name string
	}](m.Ailment)
	if err != nil {
		return nil, err
	}

	categoryJson, err := scripts.FollowNamedResource[struct {
		Id   int
		Name string
	}](m.Category)
	if err != nil {
		return nil, err
	}

	meta := &game.MoveMeta{
		Ailment:  ailmentJson,
		Category: categoryJson,

		AilmentChance: m.AilmentChance,
		FlinchChance:  m.FlinchChance,
		StatChance:    m.StatChance,

		MinHits:  m.MinHits,
		MaxHits:  m.MaxHits,
		MinTurns: m.MinTurns,
		MaxTurns: m.MaxTurns,

		Drain:         m.Drain,
		Healing:       m.Healing,
		CritRateBonus: m.CritRateBonus,
	}

	return meta, nil
}

type StatChangePre struct {
	Change int
	Stat   struct {
		Name string
		Url  string
	}
}

// TODO: Follow all NamedApiResources and return their actual value
type MoveFullPre struct {
	Accuracy      int
	DamageClass   game.NamedApiResource `json:"damage_class"`
	EffectChance  int
	EffectEntries []struct {
		Effect      string
		Language    game.NamedApiResource
		ShortEffect string `json:"short_effect"`
	} `json:"effect_entries"`
	LearnedByPokemon []game.NamedApiResource `json:"learned_by_pokemon"`
	Meta             *MoveMetaPre
	Name             string
	Power            int
	PP               int
	Priority         int
	StatChanges      []StatChangePre `json:"stat_changes"`
	Target           game.NamedApiResource
	Type             game.NamedApiResource
}

func (m *MoveFullPre) ToFullMeta() (*game.MoveFull, error) {
	damageClassJson, err := scripts.FollowNamedResource[struct {
		Name string
	}](m.DamageClass)
	if err != nil {
		return nil, err
	}

	targetJson, err := scripts.FollowNamedResource[struct {
		Id           int
		Name         string
		Descriptions []struct {
			Description string
			Language    game.NamedApiResource
		}
	}](m.Target)
	if err != nil {
		return nil, err
	}

	var effect game.EffectEntry
	for _, effectEntry := range m.EffectEntries {
		if effectEntry.Language.Name == "en" {
			effect.Effect = effectEntry.Effect
			effect.ShortEffect = effectEntry.ShortEffect
		}
	}

	target := game.Target{
		Id:   targetJson.Id,
		Name: targetJson.Name,
	}

	var meta *game.MoveMeta

	if m.Meta != nil {
		meta1, err := m.Meta.ToFullMeta()
		if err != nil {
			return nil, err
		}

		meta = meta1
	}

	titleCaser := cases.Title(language.English)

	move := &game.MoveFull{
		DamageClass: damageClassJson.Name,
		EffectEntry: effect,
		Target:      target,
		Type:        titleCaser.String(m.Type.Name),

		Accuracy:         m.Accuracy,
		EffectChance:     m.EffectChance,
		LearnedByPokemon: m.LearnedByPokemon,
		Meta:             meta,
		Name:             m.Name,
		Power:            m.Power,
		PP:               m.PP,
		Priority:         m.Priority,
		StatChanges: lo.Map(m.StatChanges, func(statPre StatChangePre, _ int) game.StatChange {
			return game.StatChange{
				Change:   statPre.Change,
				StatName: statPre.Stat.Name,
			}
		}),
	}

	return move, nil
}

func unwrap(err error) {
	if err != nil {
		panic(err)
	}
}

// Fetches and downloads all move data from pokeapi and does extra requests
// for important move data like damage classes and effect descriptions
func main() {
	moveUrl := "https://pokeapi.co/api/v2/move/?offset=0&limit=1000"

	type Response struct {
		Count    int
		Next     *string // Pointers because they can be nil
		Previous *string
		Results  []game.NamedApiResource
	}

	fullresponseJson := new(Response)

	log.Println("Grabbing moves from pokeapi")
	// Grab all moves from pokeapi
	for {
		response, err := http.Get(moveUrl)
		unwrap(err)

		bytes, err := io.ReadAll(response.Body)
		unwrap(err)

		tempResponseJson := new(Response)

		err = json.Unmarshal(bytes, tempResponseJson)
		unwrap(err)

		fullresponseJson.Results = append(fullresponseJson.Results, tempResponseJson.Results...)

		if tempResponseJson.Next != nil {
			moveUrl = *tempResponseJson.Next
		} else {
			break
		}
	}

	log.Printf("Got %d moves\n", len(fullresponseJson.Results))
	allMoves := make([]any, 0, len(fullresponseJson.Results))

	// Maps moves that a pokemon learn
	moveMap := make(map[string][]string)

	// Query the data for all moves and add them to the array
	for _, moveStub := range fullresponseJson.Results {
		log.Printf("Querying Move: %s @ %s\n", moveStub.Name, moveStub.Url)

		url := moveStub.Url
		response, err := http.Get(url)
		unwrap(err)

		bytes, err := io.ReadAll(response.Body)
		unwrap(err)

		movePreJson := new(MoveFullPre)
		err = json.Unmarshal(bytes, movePreJson)
		unwrap(err)

		moveJson, err := movePreJson.ToFullMeta()
		unwrap(err)

		for _, pokemon := range moveJson.LearnedByPokemon {
			pokeName := strings.ToLower(pokemon.Name)
			moveName := strings.ToLower(moveJson.Name)
			moveMap[pokeName] = append(moveMap[pokeName], moveName)
		}

		// Gross hack to get rid of LearnedByPokemon info
		move := game.Move{
			Accuracy:     moveJson.Accuracy,
			DamageClass:  moveJson.DamageClass,
			EffectChance: movePreJson.EffectChance,
			EffectEntry:  moveJson.EffectEntry,
			Meta:         moveJson.Meta,
			Name:         moveJson.Name,
			Power:        moveJson.Power,
			PP:           moveJson.PP,
			Priority:     moveJson.Priority,
			StatChanges:  moveJson.StatChanges,
			Target:       moveJson.Target,
			Type:         moveJson.Type,
		}

		// log.Printf("Move: %+v\n", moveJson)
		allMoves = append(allMoves, move)
	}

	movesFileName := "./data/moves.json"
	movesMapName := "./data/movesMap.json"

	os.Remove(movesFileName)
	os.Remove(movesMapName)

	log.Printf("Logging to file %s\n", movesFileName)

	movesJson, err := json.Marshal(allMoves)
	unwrap(err)

	movesMapJson, err := json.Marshal(moveMap)
	unwrap(err)

	log.Printf("Logging moves map to file %s\n", movesMapName)

	f, err := os.Create(movesFileName)
	unwrap(err)
	defer f.Close()

	f.Write(movesJson)

	f2, err := os.Create(movesMapName)
	unwrap(err)
	defer f2.Close()

	f2.Write(movesMapJson)
}
