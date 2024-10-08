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

type Ability struct {
	Name    string
	Pokemon []struct {
		Pokemon struct {
			Name string
		}
	}
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

	abilites := make([]Ability, len(abilitiesNR))

	for i, nrAbility := range abilitiesNR {
		log.Printf("Getting Pokemon who have ability: %s\n", nrAbility.Name)
		ability, err := scripts.FollowNamedResource[Ability](nrAbility)
		if err != nil {
			panic(err)
		}

		abilites[i] = ability
	}

	abilityMap := make(map[string][]string)

	for _, ability := range abilites {
		for _, pokemon := range ability.Pokemon {
			pokemonAbilities, ok := abilityMap[pokemon.Pokemon.Name]

			if ok {
				abilityMap[pokemon.Pokemon.Name] = append(pokemonAbilities, ability.Name)
			} else {
				abilityMap[pokemon.Pokemon.Name] = []string{ability.Name}
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
