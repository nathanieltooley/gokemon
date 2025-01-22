package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/scripts"
)

type Response struct {
	Count    int
	Next     *string
	Previous *string
	Results  []game.NamedApiResource
}

type PreAbility struct {
	Name       string
	ForPokemon []struct {
		Pokemon struct {
			Name string
		}
		IsHidden bool `json:"is_hidden"`
	} `json:"pokemon"`
}

func main() {
	abilitiesNR := make([]game.NamedApiResource, 0)

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

		tempResponse := new(Response)

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

	abilityMap := make(map[string][]game.Ability)

	for _, nrAbility := range abilitiesNR {
		log.Printf("Getting pokemon who have ability: %s\n", nrAbility.Name)
		ability, err := scripts.FollowNamedResource[PreAbility](nrAbility)
		if err != nil {
			panic(err)
		}

		for _, pokemon := range ability.ForPokemon {
			log.Printf("--- %s : Is Hidden %v", pokemon.Pokemon.Name, pokemon.IsHidden)
			pokemonAbilities, ok := abilityMap[pokemon.Pokemon.Name]

			finalAbility := game.Ability{
				Name:     ability.Name,
				IsHidden: pokemon.IsHidden,
			}

			if ok {
				abilityMap[pokemon.Pokemon.Name] = append(pokemonAbilities, finalAbility)
			} else {
				abilityMap[pokemon.Pokemon.Name] = []game.Ability{finalAbility}
			}
		}
	}

	abilityMapJson, err := json.Marshal(abilityMap)
	if err != nil {
		panic(err)
	}

	abilityMapJsonName := "./data/abilities.json"
	os.Remove(abilityMapJsonName)

	f, err := os.Create(abilityMapJsonName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.Write(abilityMapJson)
}
