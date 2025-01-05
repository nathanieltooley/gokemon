package stateupdater

import (
	"cmp"
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

func syncState(mainState *state.GameState, newStates []state.StateUpdate) []state.StateUpdate {
	finalState := *mainState

	// get first (from end) non-empty state
	// we don't need to apply every state update, just the last one
	// the UI is what's interested in the intermediate states
	for i := len(newStates) - 1; i >= 0; i-- {
		s := newStates[i]
		if !s.Empty {
			finalState = s.State
			break
		}
	}

	// Clone here because of slices
	*mainState = finalState.Clone()

	// Ignore empty state updates
	return lo.Filter(newStates, func(s state.StateUpdate, _ int) bool {
		return !s.Empty
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

			moveDamage := game.Damage(*aiPokemon, *playerPokemon, move)
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
	if !u.playerNeedsToSwitch && !u.aiNeedsToSwitch || !u.playerNeedsToSwitch && u.aiNeedsToSwitch {
		u.Actions = append(u.Actions, u.BestAiAction(gameState))
	}

	switches := make([]state.SwitchAction, 0)
	otherActions := make([]state.Action, 0)

	states := make([]state.StateUpdate, 0)

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

	// Handle forced switches from pkm death
	if u.playerNeedsToSwitch || u.aiNeedsToSwitch {
		u.Actions = make([]state.Action, 0)

		u.playerNeedsToSwitch = false
		u.aiNeedsToSwitch = false

		gameState.Turn++

		return func() tea.Msg {
			time.Sleep(time.Second * 1)

			messages := lo.FlatMap(states, func(item state.StateUpdate, i int) []string {
				return item.Messages
			})

			log.Info().Msgf("States: %d", len(states))
			log.Info().Strs("Queued Messages", messages).Msg("")

			gameState.MessageHistory = append(gameState.MessageHistory, messages...)

			return TurnResolvedMessage{
				StateUpdates: states,
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

		switch a := a.(type) {
		case *state.AttackAction, *state.SkipAction:
			activePokemon := gameState.GetPlayer(a.Ctx().PlayerId).GetActivePokemon()
			aSpeed = activePokemon.Speed()
		default:
			return 0
		}

		switch b := b.(type) {
		case *state.AttackAction, *state.SkipAction:
			activePokemon := gameState.GetPlayer(b.Ctx().PlayerId).GetActivePokemon()
			bSpeed = activePokemon.Speed()
		default:
			return 0
		}

		log.Debug().
			Int("aPlayer", a.Ctx().PlayerId).
			Int("bPlayer", b.Ctx().PlayerId).
			Int("aSpeed", aSpeed).
			Int("bSpeed", bSpeed).
			Int("comp", cmp.Compare(aSpeed, bSpeed)).
			Msg("sort debug")

		return cmp.Compare(aSpeed, bSpeed)
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
			pokemon.CanAttackThisTurn = !pokemon.SwitchedInThisTurn

			if !pokemon.Alive() {
				return
			}

			// Skip attack with para
			if pokemon.Status == game.STATUS_PARA {
				para := state.NewParaAction(a.Ctx().PlayerId)
				states = append(states, syncState(gameState, para.UpdateState(*gameState))...)
			}

			// Skip attack with sleep
			if pokemon.Status == game.STATUS_SLEEP {
				sleep := state.NewSleepAction(a.Ctx().PlayerId)
				states = append(states, syncState(gameState, sleep.UpdateState(*gameState))...)
			}

			// Skip attack with frozen
			if pokemon.Status == game.STATUS_FROZEN {
				frozen := state.NewFrozenAction(a.Ctx().PlayerId)
				states = append(states, syncState(gameState, frozen.UpdateState(*gameState))...)
			}

			// Skip attack with confusion
			if pokemon.ConfusionCount > 0 {
				conf := state.NewConfusionAction(a.Ctx().PlayerId)
				states = append(states, syncState(gameState, conf.UpdateState(*gameState))...)

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
				StateUpdates:  states,
			}
		}
	}

	if !gameState.OpposingPlayer.GetActivePokemon().Alive() {
		u.aiNeedsToSwitch = true
		return func() tea.Msg {
			time.Sleep(artificalDelay)

			return ForceSwitchMessage{
				ForThisPlayer: false,
				StateUpdates:  states,
			}
		}
	}

	applyBurn := func(playerId int) {
		burn := state.NewBurnAction(playerId)
		states = append(states, syncState(gameState, burn.UpdateState(*gameState))...)
	}

	applyPoison := func(playerId int) {
		poison := state.NewPoisonAction(playerId)
		states = append(states, syncState(gameState, poison.UpdateState(*gameState))...)
	}

	applyToxic := func(playerId int) {
		poison := state.NewToxicAction(playerId)
		states = append(states, syncState(gameState, poison.UpdateState(*gameState))...)
	}

	switch gameState.LocalPlayer.GetActivePokemon().Status {
	case game.STATUS_BURN:
		applyBurn(state.PLAYER)
	case game.STATUS_POISON:
		applyPoison(state.PLAYER)
	case game.STATUS_TOXIC:
		applyToxic(state.PLAYER)
	}

	switch gameState.OpposingPlayer.GetActivePokemon().Status {
	case game.STATUS_BURN:
		applyBurn(state.AI)
	case game.STATUS_POISON:
		applyPoison(state.AI)
	case game.STATUS_TOXIC:
		applyToxic(state.AI)
	}

	messages := lo.FlatMap(states, func(item state.StateUpdate, i int) []string {
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
			StateUpdates: states,
		}
	}
}

func (u *LocalUpdater) SendAction(action state.Action) {
	u.Actions = append(u.Actions, action)
}

type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		StateUpdates  []state.StateUpdate
	}
	TurnResolvedMessage struct {
		StateUpdates []state.StateUpdate
	}
	GameOverMessage struct {
		ForThisPlayer bool
	}
)
