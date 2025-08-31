package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nathanieltooley/gokemon/golurk"
)

type AbilityResponse struct {
	Count    int
	Next     *string
	Previous *string
	Results  []golurk.NamedApiResource
}

type PreAbility struct {
	Name       string
	Generation golurk.NamedApiResource
	ForPokemon []struct {
		Pokemon struct {
			Name string
		}
		IsHidden bool `json:"is_hidden"`
	} `json:"pokemon"`
}

type Generation struct {
	Id   int
	Name string
}

func abilityMain(abilityMapJsonName string) {
	generationLimit := flag.Int("gen", 0, "Limits abilities to before and in the generation provided")
	flag.Parse()

	abilitiesNR := make([]golurk.NamedApiResource, 0)

	url := "https://pokeapi.co/api/v2/ability?offset=0&limit=1000"
	for {
		response, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		responseBytes, err := io.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}

		tempResponse := new(AbilityResponse)

		err = json.Unmarshal(responseBytes, tempResponse)
		if err != nil {
			panic(err)
		}

		abilitiesNR = append(abilitiesNR, tempResponse.Results...)

		if tempResponse.Next == nil {
			break
		} else {
			url = *tempResponse.Next
		}
	}

	abilityMap := make(map[string][]golurk.Ability)

	for _, nrAbility := range abilitiesNR {
		log.Printf("Getting pokemon who have ability: %s\n", nrAbility.Name)
		ability, err := FollowNamedResource[PreAbility](nrAbility)
		if err != nil {
			panic(err)
		}

		// Skip abilities after a certain generation
		if *generationLimit > 0 {
			gen, err := FollowNamedResource[Generation](ability.Generation)
			if err != nil {
				panic(err)
			}

			if gen.Id > *generationLimit {
				log.Printf("Skipping ability. Gen higher than limit: %d > %d", gen.Id, *generationLimit)
				continue
			}
		}

		for _, pokemon := range ability.ForPokemon {
			log.Printf("--- %s : Is Hidden %v", pokemon.Pokemon.Name, pokemon.IsHidden)
			pokemonAbilities, ok := abilityMap[pokemon.Pokemon.Name]

			finalAbility := golurk.Ability{
				Name:     ability.Name,
				IsHidden: pokemon.IsHidden,
			}

			if ok {
				abilityMap[pokemon.Pokemon.Name] = append(pokemonAbilities, finalAbility)
			} else {
				abilityMap[pokemon.Pokemon.Name] = []golurk.Ability{finalAbility}
			}
		}
	}

	abilityMapJson, err := json.Marshal(abilityMap)
	if err != nil {
		panic(err)
	}

	os.Remove(abilityMapJsonName)

	f, err := os.Create(abilityMapJsonName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.Write(abilityMapJson)
}
