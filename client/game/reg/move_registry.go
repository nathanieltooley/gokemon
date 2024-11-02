package reg

import (
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
)

type MoveRegistry struct {
	// TODO: Maybe make this a map
	MoveList []game.MoveFull
	MoveMap  map[string][]string
}

func (m *MoveRegistry) GetMove(name string) *game.MoveFull {
	for _, move := range m.MoveList {
		if move.Name == name {
			return &move
		}
	}

	return nil
}

func (m *MoveRegistry) GetFullMovesForPokemon(pokemonName string) []*game.MoveFull {
	pokemonLowerName := strings.ToLower(pokemonName)
	moves := m.MoveMap[pokemonLowerName]
	movesFull := make([]*game.MoveFull, 0, len(moves))

	for _, moveName := range moves {
		movesFull = append(movesFull, m.GetMove(moveName))
	}

	return movesFull
}
