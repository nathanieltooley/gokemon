package game

type GameState struct {
	localPlayer    Player
	opposingPlayer Player
	turn           bool
}

type Player struct {
	Name            string
	Pokes           [6]*Pokemon
	ActivePokeIndex uint8
}
