package core

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/nathanieltooley/gokemon/client/game/core"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// StateEvent represents a "single" change in GameState.
// Single here meaning a high-level of single but should multiple "things" happening in a single event
// should be strongly related.
//
// StateEvents are separate from stateCore.Actions in that Events are the low level changes of state and Actions
// represent higher level changes a user can make that are made of Events
type StateEvent interface {
	// Update will update GameState in some way. Follow-up events caused by this update are returned
	// and should be handled DIRECTLY after this state event. The second value is a list of messages to be displayed for the event.
	Update(*GameState) ([]StateEvent, []string)
}

type SwitchEvent struct {
	SwitchIndex int
	PlayerIndex int
}

func (event SwitchEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	player, opposingPlayer := getPlayerPair(gameState, event.PlayerIndex)
	newActivePkm := player.GetPokemon(event.SwitchIndex)

	log.Info().Msgf("%s switches to %s", player.Name, newActivePkm.Nickname)

	// TODO: OOB Check
	player.ActivePokeIndex = event.SwitchIndex

	followUpEvents := make([]StateEvent, 0)

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == core.STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
		log.Info().Msgf("%s had their toxic count reset to 1", newActivePkm.Nickname)
	}

	// --- Activate Abilities
	switch newActivePkm.Ability.Name {
	case "drizzle":
		followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: core.WEATHER_RAIN})
	case "sand-stream":
		followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: core.WEATHER_SANDSTORM})
	case "drought":
		followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: core.WEATHER_SUN})
	case "intimidate":
		opPokemon := opposingPlayer.GetActivePokemon()
		if opPokemon.Ability.Name != "oblivious" && opPokemon.Ability.Name != "own-tempo" && opPokemon.Ability.Name != "inner-focus" {
			followUpEvents = append(followUpEvents, NewStatChangeEvent(InvertPlayerIndex(event.PlayerIndex), core.STAT_ATTACK, -1, 100))
		}
	case "natural-cure":
		newActivePkm.Status = core.STATUS_NONE
		followUpEvents = append(followUpEvents, SimpleAbilityActivationEvent(gameState, event.PlayerIndex))
	case "trace":
		opposingPokemon := opposingPlayer.GetActivePokemon()
		newActivePkm.Ability = opposingPokemon.Ability

		followUpEvents = append(followUpEvents, SimpleAbilityActivationEvent(gameState, event.PlayerIndex))
		followUpEvents = append(followUpEvents, NewFmtMessageEvent("%s gained %s's ability: %s", newActivePkm.Nickname, opposingPokemon.Nickname, opposingPokemon.Ability.Name))
	}

	newActivePkm.SwitchedInThisTurn = true
	newActivePkm.CanAttackThisTurn = false

	messages := make([]string, 0)
	if gameState.Turn == 0 {
		messages = append(messages, fmt.Sprintf("%s sent in %s!", player.Name, newActivePkm.Nickname))
	} else {
		messages = append(messages, fmt.Sprintf("%s switched to %s!", player.Name, newActivePkm.Nickname))
	}

	log.Debug().Strs("switchEventMessages", messages).Msg("")

	return followUpEvents, messages
}

type AttackEvent struct {
	AttackerID int
	MoveID     int
}

func (event AttackEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	attacker, defender := getPlayerPair(gameState, event.AttackerID)
	defenderInt := InvertPlayerIndex(event.AttackerID)

	attackPokemon := attacker.GetActivePokemon()
	defPokemon := defender.GetActivePokemon()

	var move core.Move
	var moveVars core.BattleMove
	var pp int

	if event.MoveID == -1 {
		move = struggleMove
		pp = 1
	} else {
		// TODO: Make sure a.AttackerMove is between 0 -> 3
		move = attackPokemon.Moves[event.MoveID]
		moveVars = attackPokemon.InGameMoveInfo[event.MoveID]
		pp = moveVars.PP
	}

	// TODO: hard to test but would be nice to at some point
	if attackPokemon.Ability.Name == "serene-grace" {
		move.EffectChance *= 2
	}

	events := make([]StateEvent, 0)
	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s used %s", attackPokemon.Nickname, move.Name))

	accuracyCheck := global.GokeRand.IntN(100)

	moveAccuracy := move.Accuracy
	if moveAccuracy == 0 {
		moveAccuracy = 100
	}

	if attackPokemon.Ability.Name == "hustle" && move.DamageClass != "status" {
		moveAccuracy = int(math.Round(float64(moveAccuracy) * .80))
	}

	var effectiveEvasion float32 = 1.0
	if lo.Contains(targetsAffectedByEvasion[0:], move.Target.Name) {
		effectiveEvasion = defPokemon.Evasion()
	}

	accuracy := int(float32(moveAccuracy) * (attackPokemon.Accuracy() * effectiveEvasion))

	if gameState.Weather == core.WEATHER_SANDSTORM && defPokemon.Ability.Name == "sand-veil" {
		accuracy = int(float32(accuracy) * 0.8)
	} else if attackPokemon.Ability.Name == "compound-eyes" && move.Meta.Category.Name != "ohko" {
		accuracy = int(float32(accuracy) * 1.3)
	}

	if accuracyCheck < accuracy && pp > 0 {
		attackActionLogger().Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", accuracy).Msg("Check passed")

		defImmune := false

		// TODO: This doesn't activate through protect!
		if move.Type == core.TYPENAME_ELECTRIC && defPokemon.Ability.Name == "volt-absorb" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		// TODO: This doesn't activate through protect!
		if move.Type == core.TYPENAME_WATER && defPokemon.Ability.Name == "water-absorb" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		// TODO: This doesn't activate through protect or while frozen!
		// TODO: The boost doesn't pass with baton-pass!
		if move.Type == core.TYPENAME_FIRE && defPokemon.Ability.Name == "flash-fire" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		if defPokemon.Ability.Name == "damp" {
			if slices.Contains(core.EXPLOSIVE_MOVES, move.Name) {
				events = append(events,
					AbilityActivationEvent{
						CustomMessage: fmt.Sprintf("%s prevented %s from activating!", defPokemon.Nickname, move.Name),
					},
				)

				defImmune = true
			}
		}

		if !defImmune {
			// TODO: handle these categories
			// - swagger
			// - unique
			switch move.Meta.Category.Name {
			case "damage", "damage+heal":
				events = append(events, damageMoveHandler(*gameState, *attackPokemon, event.AttackerID, *defPokemon, defenderInt, move)...)
			case "ailment":
				events = append(events, ailmentHandler(*gameState, *defPokemon, defenderInt, move)...)
			case "damage+ailment":
				events = append(events, damageMoveHandler(*gameState, *attackPokemon, event.AttackerID, *defPokemon, defenderInt, move)...)
				events = append(events, ailmentHandler(*gameState, *defPokemon, defenderInt, move)...)
			case "net-good-stats":
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					// since its "net-good-stats", the stat change always has to benefit the user
					affectedPokemonIndex := event.AttackerID
					if statChange.Change < 0 {
						affectedPokemonIndex = InvertPlayerIndex(affectedPokemonIndex)
					}

					events = append(events, NewStatChangeEvent(affectedPokemonIndex, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			// Damages and then CHANGES the targets stats
			case "damage+lower":
				events = append(events, damageMoveHandler(*gameState, *attackPokemon, event.AttackerID, *defPokemon, defenderInt, move)...)
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					events = append(events, NewStatChangeEvent(defenderInt, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			// Damages and then CHANGES the user's stats
			// this is different from what pokeapi says (raises instead of changes)
			// and this is important because moves like draco-meteor and overheat
			// lower the user's stats but are in this category
			case "damage+raise":
				events = append(events, damageMoveHandler(*gameState, *attackPokemon, event.AttackerID, *defPokemon, defenderInt, move)...)
				lo.ForEach(move.StatChanges, func(statChange core.StatChange, _ int) {
					events = append(events, NewStatChangeEvent(event.AttackerID, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			case "heal":
				events = append(events, healHandler(gameState, event.AttackerID, move))
			case "ohko":
				events = append(events, ohkoHandler(gameState, *attackPokemon, *defPokemon, defenderInt)...)
			case "force-switch":
				events = append(events, forceSwitchHandler(gameState, defender, defenderInt)...)
			default:
				attackActionLogger().Warn().Msgf("Move, %s (%s category), has no handler!!!", move.Name, move.Meta.Category.Name)
			}
		}

		flinchChance := move.Meta.FlinchChance
		if flinchChance == 0 && attackPokemon.Ability.Name == "stench" {
			flinchChance = 10
		}

		if defPokemon.Ability.Name == "inner-focus" {
			flinchChance = 0
			events = append(events, SimpleAbilityActivationEvent(gameState, defenderInt))
		}

		if global.GokeRand.IntN(100) < flinchChance {
			events = append(events, FlinchEvent{PlayerIndex: defenderInt})
		}

		ppModifier := 1

		// -1 is struggle
		if event.MoveID != -1 {
			// TODO: this check will have to change for double battles (any opposing pokemon not the defending)
			if defPokemon.Ability.Name == "pressure" {
				ppModifier = 2
			}

			attackPokemon.InGameMoveInfo[event.MoveID].PP = pp - ppModifier
		}
	} else {
		log.Debug().Int("accuracyCheck", accuracyCheck).Int("Accuracy", accuracy).Msg("Check failed")
		messages = append(messages, fmt.Sprintf("%s missed their attack!", attackPokemon.Nickname))
	}

	return events, messages
}

type WeatherEvent struct {
	NewWeather int
}

var weatherMessageMap = map[int]string{
	core.WEATHER_NONE:      "The weather has returned to normal",
	core.WEATHER_RAIN:      "It started to rain!",
	core.WEATHER_SUN:       "The sunlight turned harsh!",
	core.WEATHER_SANDSTORM: "A sandstorm kicked up!",
}

func (event WeatherEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	gameState.Weather = event.NewWeather

	return nil, []string{weatherMessageMap[event.NewWeather]}
}

type StatChangeEvent struct {
	Chance      int
	StatName    string
	Change      int
	PlayerIndex int
}

func NewStatChangeEvent(playerIndex int, statName string, change int, chance int) StatChangeEvent {
	return StatChangeEvent{PlayerIndex: playerIndex, StatName: statName, Change: change, Chance: chance}
}

func (event StatChangeEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	statCheck := global.GokeRand.IntN(100)
	if event.Chance == 0 {
		event.Change = 100
	}

	if statCheck < event.Chance {
		log.Info().Int("statChance", event.Chance).Int("statCheck", statCheck).Msg("Stat change did pass")

		pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

		// sorry
		switch event.StatName {
		case core.STAT_ATTACK:
			pokemon.Attack.ChangeStat(event.Change)
		case core.STAT_DEFENSE:
			pokemon.Def.ChangeStat(event.Change)
		case core.STAT_SPATTACK:
			pokemon.SpAttack.ChangeStat(event.Change)
		case core.STAT_SPDEF:
			pokemon.SpDef.ChangeStat(event.Change)
		case core.STAT_SPEED:
			pokemon.RawSpeed.ChangeStat(event.Change)
		case core.STAT_ACCURACY:
			if pokemon.Ability.Name == "keen-eye" || pokemon.Ability.Name == "illuminate" {
				return []StateEvent{
					SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				}, []string{fmt.Sprintf("%s's accuracy cannot be lowered", pokemon.Nickname)}
			}

			pokemon.ChangeAccuracy(event.Change)
		case core.STAT_EVASION:
			pokemon.ChangeEvasion(event.Change)
		}

		absChange := int(math.Abs(float64(event.Change)))
		var message []string = nil

		if event.Change > 0 {
			message = []string{fmt.Sprintf("%s's %s increased by %d stages!", pokemon.Nickname, event.StatName, absChange)}
		} else {
			message = []string{fmt.Sprintf("%s's %s decreased by %d stages!", pokemon.Nickname, event.StatName, absChange)}
		}

		return nil, message
	} else {
		log.Info().Int("statChance", event.Chance).Int("statCheck", statCheck).Msg("Stat change did not pass")
		return nil, nil
	}
}

type AilmentEvent struct {
	PlayerIndex int
	Ailment     int
}

var ailmentApplicationMessages = map[int]string{
	core.STATUS_NONE:   "%s has been cured of it's afflictions!",
	core.STATUS_SLEEP:  "%s has fallen asleep!",
	core.STATUS_PARA:   "%s has been paralyzed!",
	core.STATUS_FROZEN: "%s has been frozen!",
	core.STATUS_BURN:   "%s has been burned!",
	core.STATUS_POISON: "%s has been poisoned!",
	core.STATUS_TOXIC:  "%s has been badly poisoned!",
}

func (event AilmentEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	// If pokemon already has ailment, return early
	if pokemon.Status != core.STATUS_NONE {
		return nil, []string{ailmentApplicationMessages[core.STATUS_NONE]}
	}

	// Pre-Ailment checks
	switch event.Ailment {
	case core.STATUS_PARA:
		if pokemon.Ability.Name == "limber" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be paralyzed", pokemon.Nickname),
			}, nil
		}
	// Set how many turns the pokemon is asleep for
	case core.STATUS_SLEEP:
		cantSleepMsg := fmt.Sprintf("%s cannot fall asleep", pokemon.Nickname)
		if pokemon.Ability.Name == "insomnia" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewMessageEvent(cantSleepMsg),
			}, nil
		}

		if pokemon.Ability.Name == "vital-spirit" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewMessageEvent(cantSleepMsg),
			}, nil
		}

		randTime := global.GokeRand.IntN(2) + 1

		if pokemon.Ability.Name == "early-bird" {
			randTime = int(math.Floor(float64(randTime) / 2.0))
		}

		pokemon.SleepCount = randTime
		log.Debug().Msgf("%s is now asleep for %d turns", pokemon.Nickname, pokemon.SleepCount)
	case core.STATUS_BURN:
		if pokemon.Ability.Name == "water-veil" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be burned", pokemon.Nickname),
			}, nil
		}
	case core.STATUS_POISON:
		if pokemon.Ability.Name == "immunity" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be poisoned", pokemon.Nickname),
			}, nil
		}
	case core.STATUS_FROZEN:
		if pokemon.Ability.Name == "magma-armor" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be frozen", pokemon.Nickname),
			}, nil
		}
	case core.STATUS_TOXIC:
		// Block toxic with immunity
		if pokemon.Ability.Name == "immunity" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be poisoned", pokemon.Nickname),
			}, nil
		}

		// otherwise init toxic count
		pokemon.ToxicCount = 1
	}

	pokemon.Status = event.Ailment
	return nil, []string{fmt.Sprintf(ailmentApplicationMessages[event.Ailment], pokemon.Nickname)}
}

// AbilityActivationEvent occurs when an ability is activated. This can be just the message that an ability has activated
// or the effects from the ability can also occur here. The idea behind this event is to put as many
// state changing ability actions here. Basically, an change that can happen outside of its context of activation
// should happen here.
//
// For example, wonder-guard's ability does not activate here. That happens in the Damage function
// because Damage needs to be able to stop damage because of wonder-guard. Putting the invuln here would make no
// logistical sense.
//
// The largest downside to this system is that 2 switches have to occur, one where the event is created
// and the one in [AbilityActivationEvent.Update]
type AbilityActivationEvent struct {
	CustomMessage string
	AbilityName   string
	ActivatorInt  int
}

// SimpleAbilityActivationEvent returns an AbilityActivationEvent with no custom message.
func SimpleAbilityActivationEvent(gameState *GameState, activatorInt int) AbilityActivationEvent {
	pkm := gameState.GetPlayer(activatorInt).GetActivePokemon()
	return AbilityActivationEvent{AbilityName: pkm.Ability.Name, ActivatorInt: activatorInt}
}

func (event AbilityActivationEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	if event.ActivatorInt == 0 && event.CustomMessage != "" {
		return nil, []string{event.CustomMessage}
	}

	activatorPkm := gameState.GetPlayer(event.ActivatorInt).GetActivePokemon()

	events := make([]StateEvent, 0)
	messages := make([]string, 0)

	if event.CustomMessage == "" {
		messages = append(messages, fmt.Sprintf("%s activated their ability: %s", activatorPkm.Nickname, event.AbilityName))
	} else {
		messages = append(messages, event.CustomMessage)
	}

	// NOTE: This assumes that all abilities have met their conditions to be activated.
	switch event.AbilityName {
	case "flash-fire":
		activatorPkm.FlashFire = true

		messages = []string{fmt.Sprintf("%s boosted it's fire-type attacks!", activatorPkm.Nickname)}
	case "lightning-rod":
		events = append(events, NewStatChangeEvent(event.ActivatorInt, core.STAT_SPATTACK, 1, 100))
	case "volt-absorb", "water-absorb":
		events = append(events, HealPercEvent{HealPerc: .25, PlayerIndex: event.ActivatorInt})
	case "speed-boost":
		events = append(events, NewStatChangeEvent(event.ActivatorInt, core.STAT_SPEED, 1, 100))
	case "rain-dish":
		events = append(events, HealPercEvent{HealPerc: 1.0 / 16.0, PlayerIndex: event.ActivatorInt})

		messages = []string{fmt.Sprintf("%s was healed by the rain!", activatorPkm.Nickname)}
	case "shed-skin":
		// TODO: actually test this ability
		activatorPkm.Status = core.STATUS_NONE

		messages = []string{fmt.Sprintf("%s shed it's skin!", activatorPkm.Nickname)}
	}

	return events, messages
}

type DamageEvent struct {
	Damage         uint
	PlayerIndex    int
	SupressMessage bool
	Crit           bool
}

func (event DamageEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.Damage(event.Damage)

	damagePercent := 100 * (float64(event.Damage) / float64(pokemon.MaxHp))

	messages := []string{
		fmt.Sprintf("%s took %d%% damage!", pokemon.Nickname, int(damagePercent)),
	}

	if event.Crit {
		messages = append(messages, "It critically hit!")
	}

	if event.SupressMessage {
		return nil, nil
	} else {
		return nil, messages
	}
}

type HealEvent struct {
	Heal        uint
	PlayerIndex int
}

func (event HealEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.Heal(event.Heal)

	healPerc := 100 * (float64(event.Heal) / float64(pokemon.MaxHp))
	messages := []string{
		fmt.Sprintf("%s healed %d%% of their health!", pokemon.Nickname, int(healPerc)),
	}

	return nil, messages
}

type HealPercEvent struct {
	PlayerIndex int
	HealPerc    float64
}

func (event HealPercEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.HealPerc(event.HealPerc)

	heal := 100 * event.HealPerc

	return nil, []string{
		fmt.Sprintf("%s healed by %d%%!", pokemon.Nickname, int(heal)),
	}
}

type BurnEvent struct {
	PlayerIndex int
}

func (event BurnEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	if pokemon.Alive() {
		damage := pokemon.MaxHp / 16
		return []StateEvent{DamageEvent{Damage: damage, PlayerIndex: event.PlayerIndex}}, []string{
			fmt.Sprintf("%s is burned!", pokemon.Nickname),
		}
	}

	return nil, nil
}

type PoisonEvent struct {
	PlayerIndex int
}

func (event PoisonEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	damage := pokemon.MaxHp / 8
	return []StateEvent{DamageEvent{Damage: damage, PlayerIndex: event.PlayerIndex}}, []string{
		fmt.Sprintf("%s is poisoned!", pokemon.Nickname),
	}
}

type ToxicEvent struct {
	PlayerIndex int
}

func (event ToxicEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	damage := (pokemon.MaxHp / 16) * uint(pokemon.ToxicCount)
	log.Info().Int("toxicCount", pokemon.ToxicCount).Uint("damage", damage).Msg("toxic event")

	pokemon.ToxicCount++

	return []StateEvent{DamageEvent{Damage: damage, PlayerIndex: event.PlayerIndex}}, []string{
		fmt.Sprintf("%s is badly poisoned!", pokemon.Nickname),
	}
}

type FrozenEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event FrozenEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	thawChance := .20
	thawCheck := global.GokeRand.Float64()

	message := ""

	// pokemon stays frozen
	if thawCheck > thawChance {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check failed")
		message = fmt.Sprintf("%s is frozen and cannot move", pokemon.Nickname)

		pokemon.CanAttackThisTurn = false
	} else {
		log.Info().Float64("thawCheck", thawCheck).Msg("Thaw check succeeded!")
		message = fmt.Sprintf("%s thawed out!", pokemon.Nickname)

		// No need for a new event really
		pokemon.Status = core.STATUS_NONE
		pokemon.CanAttackThisTurn = true
	}

	if pokemon.CanAttackThisTurn {
		return []StateEvent{event.FollowUpAttackEvent}, []string{message}
	}

	return nil, []string{message}
}

type ParaEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event ParaEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	paraChance := 0.5
	paraCheck := global.GokeRand.Float64()

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s is paralyzed.", pokemon.Nickname))

	if paraCheck > paraChance {
		// don't get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check passed")
		return []StateEvent{event.FollowUpAttackEvent}, messages
	} else {
		// do get para'd
		log.Info().Float64("paraCheck", paraCheck).Msg("Para Check failed")
		pokemon.CanAttackThisTurn = false

		messages = append(messages, fmt.Sprintf("%s is paralyzed and cannot move.", pokemon.Nickname))
	}

	return nil, messages
}

type FlinchEvent struct {
	PlayerIndex int
}

func (event FlinchEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.CanAttackThisTurn = false

	return nil, []string{fmt.Sprintf("%s flinched and cannot move!", pokemon.Nickname)}
}

type SleepEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event SleepEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	message := ""

	// Sleep is over
	if pokemon.SleepCount <= 0 {
		pokemon.Status = core.STATUS_NONE
		message = fmt.Sprintf("%s woke up!", pokemon.Nickname)
		pokemon.CanAttackThisTurn = true
	} else {
		message = fmt.Sprintf("%s is asleep", pokemon.Nickname)
		pokemon.CanAttackThisTurn = false
	}

	if pokemon.CanAttackThisTurn {
		return []StateEvent{event.FollowUpAttackEvent}, []string{message}
	}

	pokemon.SleepCount--

	return nil, []string{message}
}

type ApplyConfusionEvent struct {
	PlayerIndex int
}

func (event ApplyConfusionEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	// TODO: add message
	if pokemon.Ability.Name != "own-tempo" {
		confusionDuration := global.GokeRand.IntN(3) + 2
		pokemon.ConfusionCount = confusionDuration

		log.Info().Int("confusionCount", pokemon.ConfusionCount).Msg("confusion applied")
	}

	return nil, []string{fmt.Sprintf("%s is now confused!", pokemon.Nickname)}
}

type ConfusionEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event ConfusionEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.ConfusionCount--
	log.Debug().Int("newConfCount", pokemon.ConfusionCount).Msg("confusion lowered")

	confuseText := fmt.Sprintf("%s is confused", pokemon.Nickname)

	messages := make([]string, 0)
	messages = append(messages, confuseText)

	confChance := .33
	confCheck := global.GokeRand.Float64()

	// Exit early
	if confCheck > confChance {
		return []StateEvent{event.FollowUpAttackEvent}, messages
	}

	confMove := core.Move{
		Name:  "Confusion",
		Power: 40,
		Meta: &core.MoveMeta{
			Category: struct {
				Id   int
				Name string
			}{
				Name: "damage",
			},
		},
		DamageClass: core.DAMAGETYPE_PHYSICAL,
	}

	pokemon.CanAttackThisTurn = false
	dmg := Damage(*pokemon, *pokemon, confMove, false, core.WEATHER_NONE)

	messages = append(messages, fmt.Sprintf("%s hit itself in confusion.", pokemon.Nickname))

	events := make([]StateEvent, 0)
	events = append(events, DamageEvent{Damage: dmg, PlayerIndex: event.PlayerIndex})

	log.Info().Uint("damage", dmg).Msgf("%s hit itself in confusion", pokemon.Nickname)

	return events, messages
}

type SandstormDamageEvent struct {
	PlayerIndex int
}

var (
	sandNonDamageTypes     = []*core.PokemonType{&core.TYPE_ROCK, &core.TYPE_STEEL, &core.TYPE_GROUND}
	sandNonDamageAbilities = []string{"sand-force", "sand-rush", "sand-veil", "magic-guard", "overcoat"}
)

func (event SandstormDamageEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	if slices.Contains(sandNonDamageTypes, pokemon.Base.Type1) || slices.Contains(sandNonDamageTypes, pokemon.Base.Type2) {
		return nil, nil
	}

	if slices.Contains(sandNonDamageAbilities, pokemon.Ability.Name) {
		return nil, nil
	}

	if pokemon.Item == "safety-goggles" {
		return nil, nil
	}

	dmg := float64(pokemon.MaxHp) * (1.0 / 16.0)
	messages := []string{
		"The sandstorm rages.",
		fmt.Sprintf("%s was damaged by the sandstorm!", pokemon.Nickname),
	}
	dmgInt := uint(math.Ceil(dmg))
	return []StateEvent{
		DamageEvent{Damage: dmgInt, PlayerIndex: event.PlayerIndex, SupressMessage: true},
	}, messages
}

type TurnStartEvent struct{}

func (event TurnStartEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	// Reset turn flags
	// eventually this will have to change for double battles
	gameState.HostPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.HostPlayer.GetActivePokemon().SwitchedInThisTurn = false

	gameState.ClientPlayer.GetActivePokemon().CanAttackThisTurn = true
	gameState.ClientPlayer.GetActivePokemon().SwitchedInThisTurn = false

	return nil, nil
}

type EndOfTurnAbilityCheck struct {
	PlayerID int
}

func (event EndOfTurnAbilityCheck) Update(gameState *GameState) ([]StateEvent, []string) {
	playerPokemon := gameState.GetPlayer(event.PlayerID).GetActivePokemon()

	events := make([]StateEvent, 0)

	switch playerPokemon.Ability.Name {
	case "speed-boost":
		if !playerPokemon.SwitchedInThisTurn {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, event.PlayerID),
			)
		}
	case "rain-dish":
		events = append(events,
			SimpleAbilityActivationEvent(gameState, event.PlayerID),
		)
	case "shed-skin":
		check := global.GokeRand.Float32()
		if check <= .33 {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, event.PlayerID),
			)
		}
	}

	return events, nil
}

// MessageEvent is an event that only shows a message. No state updates occur.
type MessageEvent struct {
	Message string
}

func NewMessageEvent(message string) MessageEvent {
	return MessageEvent{Message: message}
}

func (event MessageEvent) Update(_ *GameState) ([]StateEvent, []string) {
	return nil, []string{event.Message}
}

// FmtMessageEvent is an event that only shows a message fmt.Sprintf'ed with the given arguments. All rules with fmt.Sprintf apply here
type FmtMessageEvent struct {
	Message string
	Args    []any
}

func NewFmtMessageEvent(message string, a ...any) FmtMessageEvent {
	return FmtMessageEvent{Message: message, Args: a}
}

func (event FmtMessageEvent) Update(_ *GameState) ([]StateEvent, []string) {
	return nil, []string{fmt.Sprintf(event.Message, event.Args...)}
}

type EventIter struct {
	events []StateEvent
}

func NewEventIter() EventIter {
	return EventIter{make([]StateEvent, 0)}
}

// Next updates state given the top event, adds any follow up events to the front of the queue,
// and returns the messages from that state to be shown to the user. The boolean value is true if
// there are any more events in the queue.
func (iter *EventIter) Next(state *GameState) ([]string, bool) {
	if len(iter.events) == 0 {
		return nil, false
	}

	headEvent := iter.events[0]
	log.Debug().Str("eventIterHeadType", reflect.TypeOf(headEvent).Name()).Msg("")
	followUpEvents, messages := headEvent.Update(state)

	log.Debug().Strs("eventIterMessages", messages).Msg("")

	// pop queue
	iter.events = iter.events[1:len(iter.events)]

	log.Debug().Msg("====== New Event Iter Queue ======")
	for _, event := range iter.events {
		log.Debug().Str("eventIterInQueue", reflect.TypeOf(event).Name()).Msg("")
	}

	if len(followUpEvents) != 0 {
		// create new queue with follow_up_events prepended to the front
		newQueue := make([]StateEvent, 0, len(iter.events)+len(followUpEvents))
		newQueue = append(newQueue, followUpEvents...)
		newQueue = append(newQueue, iter.events...)

		iter.events = newQueue
	}

	return messages, true
}

func (iter *EventIter) AddEvents(events []StateEvent) {
	iter.events = append(iter.events, events...)
}

func (iter EventIter) Len() int {
	return len(iter.events)
}

// getPlayerPair returns both the player with the given index as the first value and the opposing player as the second value
func getPlayerPair(gameState *GameState, activePlayerIndex int) (*Player, *Player) {
	player := gameState.GetPlayer(activePlayerIndex)
	opposingPlayer := gameState.GetPlayer(InvertPlayerIndex(activePlayerIndex))

	return player, opposingPlayer
}
