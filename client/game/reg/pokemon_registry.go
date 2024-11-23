package reg

import (
	"math/rand/v2"
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
)

type PokemonRegistry []game.BasePokemon

func (p PokemonRegistry) GetPokemonByPokedex(pkdNumber int) *game.BasePokemon {
	for _, pkm := range p {
		if pkm.PokedexNumber == uint(pkdNumber) {
			return &pkm
		}
	}

	return nil
}

func (p PokemonRegistry) GetPokemonByName(pkmName string) *game.BasePokemon {
	for _, pkm := range p {
		if strings.ToLower(pkm.Name) == strings.ToLower(pkmName) {
			return &pkm
		}
	}

	return nil
}

func (p PokemonRegistry) GetRandomPokemon() *game.BasePokemon {
	pkmIndex := rand.IntN(len(p))

	return &p[pkmIndex]
}
