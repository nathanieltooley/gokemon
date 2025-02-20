package networking

import (
	"github.com/nathanieltooley/gokemon/client/game"
)

type TeamSelectionPacket struct {
	Team          []game.Pokemon
	StartingIndex int
}
