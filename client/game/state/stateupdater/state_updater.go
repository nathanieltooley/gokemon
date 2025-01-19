package stateupdater

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type StateUpdater interface {
	// Updates the state of the game
	Update(*state.GameState) tea.Cmd

	// Sets the Host's action for this turn
	SendAction(state.Action)
}

func syncState(mainState *state.GameState, newStates []state.StateSnapshot) []state.StateSnapshot {
	finalState := *mainState

	// get first (from end) non-empty state
	// we don't need to apply every state update, just the last one
	// the UI is what's interested in the intermediate states
	for i := len(newStates) - 1; i >= 0; i-- {
		s := newStates[i]
		if !s.Empty && !s.MessagesOnly {
			finalState = s.State
			break
		}
	}

	// Clone here because of slices
	*mainState = finalState.Clone()

	return newStates
}

func cleanStateSnapshots(snaps []state.StateSnapshot) []state.StateSnapshot {
	// Ignore empty state updates
	snaps = lo.Filter(snaps, func(s state.StateSnapshot, _ int) bool {
		return !s.Empty
	})

	if len(snaps) <= 1 {
		return snaps
	}

	// if a snapshot only contains messages, append them to the last real snapshot
	previousSnap := &snaps[0]
	for i := 1; i < len(snaps); i++ {
		currentSnap := snaps[i]
		if currentSnap.MessagesOnly {
			previousSnap.Messages = append(previousSnap.Messages, currentSnap.Messages...)
		} else {
			previousSnap = &snaps[i]
		}
	}

	// Get rid of message only snapshots
	return lo.Filter(snaps, func(s state.StateSnapshot, _ int) bool {
		return !s.MessagesOnly
	})
}

type LocalUpdater struct {
	Actions []state.Action

	aiNeedsToSwitch     bool
	playerNeedsToSwitch bool
}

func (u *LocalUpdater) BestAiAction(gameState *state.GameState) state.Action {
	if gameState.OpposingPlayer.GetActivePokemon().Alive() {
		playerPokemon := gameState.LocalPlayer.GetActivePokemon()
		aiPokemon := gameState.OpposingPlayer.GetActivePokemon()

		bestMoveIndex := 0
		var bestMove *game.Move
		var bestMoveDamage uint = 0

		for i, move := range aiPokemon.Moves {
			if move == nil {
				continue
			}

			// assume no crits
			moveDamage := game.Damage(*aiPokemon, *playerPokemon, move, false)
			if moveDamage > bestMoveDamage {
				bestMoveIndex = i
				bestMove = move
				bestMoveDamage = moveDamage
			}
		}

		if bestMove == nil {
			return &state.SkipAction{}
		} else {
			return state.NewAttackAction(state.AI, bestMoveIndex)
		}

	} else {
		// Switch on death
		for i, pokemon := range gameState.OpposingPlayer.Team {
			if pokemon.Alive() {
				return state.NewSwitchAction(gameState, state.AI, i)
			}
		}
	}

	return &state.SkipAction{}
}

func (u *LocalUpdater) Update(gameState *state.GameState) tea.Cmd {
	artificalDelay := time.Second * 2

	// Do not have the AI add a new action to the list if the player is switching and the AI isnt
	// probably a better expression for this
	if !u.playerNeedsToSwitch && !u.aiNeedsToSwitch || !u.playerNeedsToSwitch && u.aiNeedsToSwitch {
		u.Actions = append(u.Actions, u.BestAiAction(gameState))
	}

	switches := make([]state.SwitchAction, 0)
	otherActions := make([]state.Action, 0)

	states := make([]state.StateSnapshot, 0)

	for _, a := range u.Actions {
		switch a := a.(type) {
		case *state.SwitchAction:
			switches = append(switches, *a)
		default:
			otherActions = append(otherActions, a)
		}
	}

	u.Actions = make([]state.Action, 0)

	// Sort switching order by speed
	slices.SortFunc(switches, func(a, b state.SwitchAction) int {
		return cmp.Compare(a.Poke.Speed(), b.Poke.Speed())
	})

	// Reverse for desc order
	slices.Reverse(switches)

	// Process switches first
	lo.ForEach(switches, func(a state.SwitchAction, i int) {
		states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
	})

	// Properly end turn after force switches are dealt with
	if u.playerNeedsToSwitch || u.aiNeedsToSwitch {
		u.Actions = make([]state.Action, 0)

		u.playerNeedsToSwitch = false
		u.aiNeedsToSwitch = false

		gameState.Turn++

		return func() tea.Msg {
			time.Sleep(time.Second * 1)

			messages := lo.FlatMap(states, func(item state.StateSnapshot, i int) []string {
				return item.Messages
			})

			log.Info().Msgf("States: %d", len(states))
			log.Info().Strs("Queued Messages", messages).Msg("")

			gameState.MessageHistory = append(gameState.MessageHistory, messages...)

			return TurnResolvedMessage{
				StateUpdates: cleanStateSnapshots(states),
			}
		}
	} else {
		log.Info().Msgf("\n\n======== TURN %d =========", gameState.Turn)
	}

	// Reset turn flags
	// eventually this will have to change for double battles
	gameState.LocalPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.LocalPlayer.GetActivePokemon().SwitchedInThisTurn = false

	gameState.OpposingPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.OpposingPlayer.GetActivePokemon().SwitchedInThisTurn = false

	// Sort Other Actions
	slices.SortFunc(otherActions, func(a, b state.Action) int {
		var aSpeed int
		var bSpeed int
		var aPriority int
		var bPriority int

		activePokemon := gameState.GetPlayer(a.Ctx().PlayerId).GetActivePokemon()
		aSpeed = activePokemon.Speed()

		switch a := a.(type) {
		case *state.AttackAction:
			move := activePokemon.Moves[a.AttackerMove]
			aPriority = move.Priority
		case *state.SkipAction:
			aPriority = -100
		default:
			return 0
		}

		activePokemon = gameState.GetPlayer(b.Ctx().PlayerId).GetActivePokemon()
		bSpeed = activePokemon.Speed()

		switch b := b.(type) {
		case *state.AttackAction:
			if b.AttackerMove < 0 || b.AttackerMove >= len(activePokemon.Moves) {
				return 0
			}

			move := activePokemon.Moves[b.AttackerMove]
			bPriority = move.Priority
		case *state.SkipAction:
			bPriority = -100
		default:
			return 0
		}

		log.Debug().
			Int("aPlayer", a.Ctx().PlayerId).
			Int("bPlayer", b.Ctx().PlayerId).
			Int("aSpeed", aSpeed).
			Int("bSpeed", bSpeed).
			Int("aPriority", aPriority).
			Int("bPriority", bPriority).
			Int("comp", cmp.Compare(aSpeed, bSpeed)).
			Int("compPriority", cmp.Compare(aPriority, bPriority)).
			Msg("sort debug")

		priorComp := cmp.Compare(aPriority, bPriority)
		speedComp := cmp.Compare(aSpeed, bSpeed)

		if priorComp == 0 {
			return speedComp
		} else {
			return priorComp
		}
	})

	// Reverse for desc order
	slices.Reverse(otherActions)

	// Process otherActions next
	lo.ForEach(otherActions, func(a state.Action, i int) {
		switch a.(type) {
		case *state.AttackAction, *state.SkipAction:
			player := gameState.GetPlayer(a.Ctx().PlayerId)

			log.Info().Int("attackIndex", i).
				Int("attackerSpeed", player.GetActivePokemon().Speed()).
				Int("attackerRawSpeed", player.GetActivePokemon().RawSpeed.CalcValue()).
				Int("attackerConfCount", player.GetActivePokemon().ConfusionCount).
				Msg("Attack state update")

			pokemon := player.GetActivePokemon()
			if pokemon.CanAttackThisTurn {
				pokemon.CanAttackThisTurn = !pokemon.SwitchedInThisTurn
			}

			if !pokemon.Alive() {
				return
			}

			// Skip attack with para
			if pokemon.Status == game.STATUS_PARA {
				states = append(states, state.ParaHandler(gameState, pokemon))
			}

			// Skip attack with sleep
			if pokemon.Status == game.STATUS_SLEEP {
				states = append(states, state.SleepHandler(gameState, pokemon))
			}

			// Skip attack with frozen
			if pokemon.Status == game.STATUS_FROZEN {
				states = append(states, state.FreezeHandler(gameState, pokemon))
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				states = append(states, state.ConfuseHandler(gameState, pokemon))
				pokemon.ConfusionCount--

				log.Debug().Int("newConfCount", pokemon.ConfusionCount).Msg("confusion turn completed")
			}

			if pokemon.Alive() && pokemon.CanAttackThisTurn {
				states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
			}
		default:
			states = append(states, syncState(gameState, a.UpdateState(*gameState))...)
		}
	})

	gameOverValue := gameState.GameOver()
	if gameOverValue == state.PLAYER {
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return GameOverMessage{
				ForThisPlayer: true,
			}
		}
	} else if gameOverValue == state.AI {
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return GameOverMessage{
				ForThisPlayer: false,
			}
		}
	}

	// Seems weird but should make sense if or when multiplayer is added
	// also these checks will have to change if double battles are added
	if !gameState.LocalPlayer.GetActivePokemon().Alive() {
		u.playerNeedsToSwitch = true
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return ForceSwitchMessage{
				ForThisPlayer: true,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
		u.aiNeedsToSwitch = true
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  cleanStateSnapshots(states),
			}
		}
	}

	applyBurn := func(pokemon *game.Pokemon) {
		states = append(states, state.BurnHandler(gameState, pokemon))
	}

	applyPoison := func(pokemon *game.Pokemon) {
		states = append(states, state.PoisonHandler(gameState, pokemon))
	}

	applyToxic := func(pokemon *game.Pokemon) {
		states = append(states, state.ToxicHandler(gameState, pokemon))
	}

	localPokemon := gameState.LocalPlayer.GetActivePokemon()
	switch localPokemon.Status {
	case game.STATUS_BURN:
		applyBurn(localPokemon)
	case game.STATUS_POISON:
		applyPoison(localPokemon)
	case game.STATUS_TOXIC:
		applyToxic(localPokemon)
	}

	opPokemon := gameState.OpposingPlayer.GetActivePokemon()
	switch opPokemon.Status {
	case game.STATUS_BURN:
		applyBurn(opPokemon)
	case game.STATUS_POISON:
		applyPoison(opPokemon)
	case game.STATUS_TOXIC:
		applyToxic(opPokemon)
	}

	messages := lo.FlatMap(states, func(item state.StateSnapshot, i int) []string {
		return item.Messages
	})

	log.Info().Msgf("States: %d", len(states))
	log.Info().Strs("Queued Messages", messages).Msg("")

	gameState.MessageHistory = append(gameState.MessageHistory, messages...)

	return func() tea.Msg {
		// Artifical Delay
		time.Sleep(artificalDelay)

		gameState.Turn++

		return TurnResolvedMessage{
			StateUpdates: cleanStateSnapshots(states),
		}
	}
}

func (u *LocalUpdater) SendAction(action state.Action) {
	u.Actions = append(u.Actions, action)
}

func endOfTurnAbilities(gameState *state.GameState, player int) []state.StateSnapshot {
	playerPokemon := gameState.GetPlayer(player).GetActivePokemon()

	states := make([]state.StateSnapshot, 0)

	abilityText := fmt.Sprintf("%s activated their ability: %s", playerPokemon.Nickname, playerPokemon.Ability.Name)

	switch playerPokemon.Ability.Name {
	// TEST: no gen 1 pkm have this ability
	case "speed-boost":
		states = append(states, state.NewMessageOnlySnapshot(abilityText))
		states = append(states, state.StatChangeHandler(gameState, playerPokemon, game.StatChange{Change: 1, StatName: "speed"}, 100))
	}

	return states
}

type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		StateUpdates  []state.StateSnapshot
	}
	TurnResolvedMessage struct {
		StateUpdates []state.StateSnapshot
	}
	GameOverMessage struct {
		ForThisPlayer bool
	}
)
