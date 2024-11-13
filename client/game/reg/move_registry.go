package reg

import (
	"strings"

	"github.com/nathanieltooley/gokemon/client/game"
)

type MoveRegistry struct {
	// TODO: Maybe make this a map
	MoveList []game.Move
	MoveMap  map[string][]string
}

func (m *MoveRegistry) GetMove(name string) *game.Move {
	for _, move := range m.MoveList {
		if move.Name == name {
			return &move
		}
	}

	return nil
}

func (m *MoveRegistry) GetFullMovesForPokemon(pokemonName string) []*game.Move {
	pokemonLowerName := strings.ToLower(pokemonName)
	moves := m.MoveMap[pokemonLowerName]
	movesFull := make([]*game.Move, 0, len(moves))

	for _, moveName := range moves {
		movesFull = append(movesFull, m.GetMove(moveName))
	}

	return movesFull
}
