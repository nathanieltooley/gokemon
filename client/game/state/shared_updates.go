package state

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/rs/zerolog/log"
)

const (
	STAT_ATTACK   = "attack"
	STAT_DEFENSE  = "defense"
	STAT_SPATTACK = "special-attack"
	STAT_SPDEF    = "special-defense"
	STAT_SPEED    = "speed"
)

func StatChangeHandler(state *GameState, pokemon *game.Pokemon, statChange game.StatChange, statChance int) StateSnapshot {
	statCheck := rand.Intn(100)
	if statChance == 0 {
		statChance = 100
	}

	statChangeState := StateSnapshot{}

	if statCheck < statChance {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did pass")
		statChangeState.Messages = append(statChangeState.Messages, ChangeStat(pokemon, statChange.StatName, statChange.Change)...)
	} else {
		log.Info().Int("statChance", statChance).Int("statCheck", statCheck).Msg("Stat change did not pass")
	}

	statChangeState.State = state.Clone()

	return statChangeState
}

func ChangeStat(pokemon *game.Pokemon, statName string, change int) []string {
	messages := make([]string, 0)

	absChange := int(math.Abs(float64(change)))
	if change > 0 {
		messages = append(messages, fmt.Sprintf("%s's %s increased by %d stages!", pokemon.Nickname, statName, absChange))
	} else {
		messages = append(messages, fmt.Sprintf("%s's %s decreased by %d stages!", pokemon.Nickname, statName, absChange))
	}

	// sorry
	switch statName {
	case STAT_ATTACK:
		pokemon.Attack.ChangeStat(change)
	case STAT_DEFENSE:
		pokemon.Def.ChangeStat(change)
	case STAT_SPATTACK:
		pokemon.SpAttack.ChangeStat(change)
	case STAT_SPDEF:
		pokemon.SpDef.ChangeStat(change)
	case STAT_SPEED:
		pokemon.RawSpeed.ChangeStat(change)
	}

	return messages
}

func FlinchHandler(state *GameState, pokemon *game.Pokemon) StateSnapshot {
	pokemon.CanAttackThisTurn = false
	return StateSnapshot{
		State:    state.Clone(),
		Messages: []string{fmt.Sprintf("%s flinched!", pokemon.Nickname)},
	}
}
