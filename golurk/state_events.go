package golurk

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/samber/lo"
)

// StateEvent represents a "single" change in GameState.
// Single here meaning a high-level of single but should multiple "things" happening in a single event
// should be strongly related.
//
// StateEvents are separate from stateActions in that Events are the low level changes of state and Actions
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
	currentPokemon := player.GetActivePokemon()
	newActivePkm := player.GetPokemon(event.SwitchIndex)

	currentPokemon.ClearStatChanges()
	currentPokemon.TauntCount = 0
	currentPokemon.InfatuationTarget = -1

	opposingPokemon := opposingPlayer.GetActivePokemon()
	switch opposingPokemon.Ability.Name {
	case "shadow-tag", "arena-trap":
		return nil, []string{fmt.Sprintf("%s could not switch out!", currentPokemon.Name())}
	case "magnet-pull":
		if currentPokemon.HasType(&TYPE_STEEL) {
			return nil, []string{fmt.Sprintf("%s could not switch out!", currentPokemon.Name())}
		}
	}

	messages := make([]string, 0)

	// If we are switching out the opposing pokemon's InfatuationTarget, remove their infatuation
	if opposingPokemon.InfatuationTarget == player.ActivePokeIndex {
		opposingPokemon.InfatuationTarget = -1
		messages = append(messages, fmt.Sprintf("%s is no longer infatuated!", opposingPokemon.Name()))
	}

	internalLogger.WithName("switch_event").Info("", "player_name", player.Name, "pokemon_name", newActivePkm.Name())

	// TODO: OOB Check
	player.ActivePokeIndex = event.SwitchIndex

	followUpEvents := make([]StateEvent, 0)

	// --- On Switch-In Updates ---
	// Reset toxic count
	if newActivePkm.Status == STATUS_TOXIC {
		newActivePkm.ToxicCount = 1
		internalLogger.WithName("switch_event").Info("pokemon switched in and reset their toxic count", "pokemon_name", newActivePkm.Name())
	}

	// --- Activate Abilities
	switch newActivePkm.Ability.Name {
	case "drizzle":
		if gameState.DisabledWeather == WEATHER_NONE {
			followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: WEATHER_RAIN})
		}
	case "sand-stream":
		if gameState.DisabledWeather == WEATHER_NONE {
			followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: WEATHER_SANDSTORM})
		}
	case "drought":
		if gameState.DisabledWeather == WEATHER_NONE {
			followUpEvents = append(followUpEvents, WeatherEvent{NewWeather: WEATHER_SUN})
		}
	case "cloud-nine", "air-lock":
		gameState.DisabledWeather = gameState.Weather
		gameState.Weather = WEATHER_NONE

		messages = append(messages, "The effects of weather disappeared")
	case "intimidate":
		opPokemon := opposingPlayer.GetActivePokemon()
		if opPokemon.Ability.Name != "oblivious" && opPokemon.Ability.Name != "own-tempo" && opPokemon.Ability.Name != "inner-focus" {
			followUpEvents = append(followUpEvents, NewStatChangeEvent(InvertPlayerIndex(event.PlayerIndex), STAT_ATTACK, -1, 100))
		}
	case "natural-cure":
		if newActivePkm.Status != STATUS_NONE {
			newActivePkm.Status = STATUS_NONE
			followUpEvents = append(followUpEvents, SimpleAbilityActivationEvent(gameState, event.PlayerIndex))
		}
	case "trace":
		opposingPokemon := opposingPlayer.GetActivePokemon()
		newActivePkm.Ability = opposingPokemon.Ability

		// manual message event used here because AbilityActivationEvent would use the new ability
		followUpEvents = append(followUpEvents, NewFmtMessageEvent("%s activated trace!"))
		followUpEvents = append(followUpEvents, NewFmtMessageEvent("%s gained %s's ability: %s", newActivePkm.Name(), opposingPokemon.Name(), opposingPokemon.Ability.Name))
	case "forecast":
		followUpEvents = append(followUpEvents, SimpleAbilityActivationEvent(gameState, event.PlayerIndex))
	}

	newActivePkm.SwitchedInThisTurn = true
	newActivePkm.CanAttackThisTurn = false
	// no matter what, pokemon should not truant on the turn they switch in
	newActivePkm.TruantShouldActivate = false

	if gameState.Turn == 0 || gameState.Turn == 1 {
		messages = append(messages, fmt.Sprintf("%s sent in %s!", player.Name, newActivePkm.Name()))
	} else {
		messages = append(messages, fmt.Sprintf("%s switched to %s!", player.Name, newActivePkm.Name()))
	}

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

	if !attackPokemon.Alive() {
		attackEventLogger().Info("attack was cancelled because they died", "pokemon_name", attackPokemon.Name())
		return nil, nil
	}

	rng := gameState.CreateRng()

	var move Move
	var moveVars BattleMove
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

	// Ideally, the player should not be able to get into this situation as the client should stop them.
	// However, in the case it fails or they cheat, the move will become struggle
	if attackPokemon.TauntCount > 0 && move.DamageClass == "status" {
		attackEventLogger().Info("Player somehow attempted to use a status move while taunted!", "move_name", move.Name)
		move = struggleMove
		pp = 1
	}

	// HACK: Bulbapedia says that if the user of taunt acts before the target, taunt only last 3 turns.
	// The idea here is that if taunt is used before the attackPokemon's turn the taunt count will be 4 and we just set it to 3 before
	// it get lowers at the end of the turn.
	// This code assumes that no other actions will be added (as switches remove taunt and skip actions are a failsafe).
	if attackPokemon.TauntCount == 4 {
		attackPokemon.TauntCount = 3
	}

	// TODO: hard to test but would be nice to at some point
	if attackPokemon.Ability.Name == "serene-grace" {
		move.EffectChance *= 2
	}

	if defPokemon.Ability.Name == "liquid-ooze" && move.Meta.Drain > 0 {
		move.Meta.Drain *= -1
	}

	if defPokemon.Ability.Name == "soundproof" && lo.Contains(SOUND_MOVES, move.Name) {
		return []StateEvent{
			SimpleAbilityActivationEvent(gameState, defenderInt),
		}, []string{fmt.Sprintf("%s is not affected by sound based moves!", defPokemon.Name())}
	}

	// TODO: hard to test but would be nice to at some point
	if (gameState.Weather == WEATHER_HAIL || gameState.Weather == WEATHER_SNOW) && move.Name == "blizzard" {
		move.Accuracy = 100
	}

	events := make([]StateEvent, 0)
	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s used %s", attackPokemon.Name(), move.Name))

	accuracyCheck := rng.IntN(100)

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

	if gameState.Weather == WEATHER_SANDSTORM && defPokemon.Ability.Name == "sand-veil" {
		accuracy = int(float32(accuracy) * 0.8)
	} else if attackPokemon.Ability.Name == "compound-eyes" && move.Meta.Category.Name != "ohko" {
		accuracy = int(float32(accuracy) * 1.3)
	}

	if accuracyCheck < accuracy && pp > 0 {
		attackEventLogger().Info("accuracy check passed", "accuracy_check", accuracyCheck, "accuracy_chance", accuracy)

		defImmune := false

		// TODO: This doesn't activate through protect!
		if move.Type == TYPENAME_ELECTRIC && defPokemon.Ability.Name == "volt-absorb" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		// TODO: This doesn't activate through protect!
		if move.Type == TYPENAME_WATER && defPokemon.Ability.Name == "water-absorb" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		// TODO: This doesn't activate through protect or while frozen!
		// TODO: The boost doesn't pass with baton-pass!
		if move.Type == TYPENAME_FIRE && defPokemon.Ability.Name == "flash-fire" {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, defenderInt),
			)

			defImmune = true
		}

		if defPokemon.Ability.Name == "damp" {
			if slices.Contains(EXPLOSIVE_MOVES, move.Name) {
				events = append(events,
					AbilityActivationEvent{
						CustomMessage: fmt.Sprintf("%s prevented %s from activating!", defPokemon.Name(), move.Name),
					},
				)

				defImmune = true
			}
		}

		// TODO: untested, lol
		if lo.Contains(CONTACT_MOVES, move.Name) {
			switch defPokemon.Ability.Name {
			case "flame-body":
				effectChance := .3
				effectCheck := rng.Float64()

				if effectCheck < effectChance {
					events = append(events, AilmentEvent{PlayerIndex: event.AttackerID, Ailment: STATUS_BURN})
				}
			case "poison-point":
				effectChance := .3
				effectCheck := rng.Float64()

				if effectCheck < effectChance {
					events = append(events, AilmentEvent{PlayerIndex: event.AttackerID, Ailment: STATUS_POISON})
				}
			case "effect-spore":
				effectCheck := rng.Float64()

				if effectCheck < 3.33 {
					events = append(events, AilmentEvent{PlayerIndex: event.AttackerID, Ailment: STATUS_POISON})
				} else if effectCheck < 6.66 {
					events = append(events, AilmentEvent{PlayerIndex: event.AttackerID, Ailment: STATUS_PARA})
				} else if effectCheck < 9.99 {
					events = append(events, AilmentEvent{PlayerIndex: event.AttackerID, Ailment: STATUS_SLEEP})
				}
			case "rough-skin":
				dmg := float64(attackPokemon.MaxHp) * (1.0 / 16.0)
				dmgInt := uint(dmg)

				events = append(events, DamageEvent{PlayerIndex: event.AttackerID, Damage: dmgInt})
			case "cute-charm":
				effectChance := .3
				effectCheck := rng.Float64()

				if effectCheck < effectChance {
					if OppositeGenders(*attackPokemon, *defPokemon) {
						events = append(events, ApplyInfatuationEvent{PlayerIndex: event.AttackerID, Target: defender.ActivePokeIndex})
					}
				}
			}
		}

		handlerContext := newAttackHandlerContext(*gameState, event.AttackerID, defenderInt, move)

		if !defImmune {
			// TODO: handle these categories
			// - swagger
			// - unique
			switch move.Meta.Category.Name {
			case "damage", "damage+heal":
				events = append(events, damageMoveHandler(handlerContext)...)
			case "ailment":
				events = append(events, ailmentHandler(handlerContext)...)
			case "damage+ailment":
				events = append(events, damageMoveHandler(handlerContext)...)
				events = append(events, ailmentHandler(handlerContext)...)
			case "net-good-stats":
				lo.ForEach(move.StatChanges, func(statChange StatChange, _ int) {
					// since its "net-good-stats", the stat change always has to benefit the user
					affectedPokemonIndex := event.AttackerID
					if statChange.Change < 0 {
						affectedPokemonIndex = InvertPlayerIndex(affectedPokemonIndex)
					}

					events = append(events, NewStatChangeEvent(affectedPokemonIndex, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			// Damages and then CHANGES the targets stats
			case "damage+lower":
				events = append(events, damageMoveHandler(handlerContext)...)
				lo.ForEach(move.StatChanges, func(statChange StatChange, _ int) {
					events = append(events, NewStatChangeEvent(defenderInt, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			// Damages and then CHANGES the user's stats
			// this is different from what pokeapi says (raises instead of changes)
			// and this is important because moves like draco-meteor and overheat
			// lower the user's stats but are in this category
			case "damage+raise":
				events = append(events, damageMoveHandler(handlerContext)...)
				lo.ForEach(move.StatChanges, func(statChange StatChange, _ int) {
					events = append(events, NewStatChangeEvent(event.AttackerID, statChange.StatName, statChange.Change, move.Meta.StatChance))
				})
			case "heal":
				events = append(events, healHandler(handlerContext))
			case "ohko":
				events = append(events, ohkoHandler(handlerContext)...)
			case "force-switch":
				events = append(events, forceSwitchHandler(handlerContext)...)
			case "unique":
				switch move.Name {
				case "taunt":
					if defPokemon.Ability.Name != "oblivious" {
						defPokemon.TauntCount = 4
					}
				default:
					attackEventLogger().Info("Unique attack has no handler!!!", "move_name", move.Name)
				}
			default:
				attackEventLogger().Info("Move has no handler!!!", "move_name", move.Name, "move_category", move.Meta.Category.Name)
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

		if rng.IntN(100) < flinchChance {
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
		attackEventLogger().Info("accuracy check failed", "accuracy_check", accuracyCheck, "accuracy_chance", accuracy, "pokemon_name", attackPokemon.Name())
		messages = append(messages, fmt.Sprintf("%s missed their attack!", attackPokemon.Name()))
	}

	attackPokemon.TruantShouldActivate = true

	return events, messages
}

type WeatherEvent struct {
	NewWeather int
}

var weatherMessageMap = map[int]string{
	WEATHER_NONE:      "The weather has returned to normal",
	WEATHER_RAIN:      "It started to rain!",
	WEATHER_SUN:       "The sunlight turned harsh!",
	WEATHER_SANDSTORM: "A sandstorm kicked up!",
}

func (event WeatherEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	gameState.Weather = event.NewWeather

	events := make([]StateEvent, 0)
	hostPoke := gameState.HostPlayer.GetActivePokemon()
	clientPoke := gameState.ClientPlayer.GetActivePokemon()

	if hostPoke.Ability.Name == "forecast" {
		events = append(events, SimpleAbilityActivationEvent(gameState, HOST))
	}

	if clientPoke.Ability.Name == "forecast" {
		events = append(events, SimpleAbilityActivationEvent(gameState, PEER))
	}

	return events, []string{weatherMessageMap[event.NewWeather]}
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
	rng := gameState.CreateRng()

	statCheck := rng.IntN(100)
	if event.Chance == 0 {
		event.Chance = 100
	}

	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	if event.Change < 0 && (pokemon.Ability.Name == "white-smoke" || pokemon.Ability.Name == "clear-body") {
		return []StateEvent{
			SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
		}, []string{fmt.Sprintf("%s's stats cannot be lowered!", pokemon.Name())}
	}

	if statCheck < event.Chance {
		internalLogger.WithName("stat_change_event").Info("stat change check passed", "stat_check", statCheck, "stat_chance", event.Chance, "pokemon_name", pokemon.Name())

		// sorry
		switch event.StatName {
		case STAT_ATTACK:
			if event.Change < 0 && pokemon.Ability.Name == "hyper-cutter" {
				return []StateEvent{
					SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				}, []string{fmt.Sprintf("%s's attack cannot be lowered!", pokemon.Name())}
			}
			pokemon.Attack.ChangeStat(event.Change)
		case STAT_DEFENSE:
			pokemon.Def.ChangeStat(event.Change)
		case STAT_SPATTACK:
			pokemon.SpAttack.ChangeStat(event.Change)
		case STAT_SPDEF:
			pokemon.SpDef.ChangeStat(event.Change)
		case STAT_SPEED:
			pokemon.RawSpeed.ChangeStat(event.Change)
		case STAT_ACCURACY:
			if event.Change < 0 && (pokemon.Ability.Name == "keen-eye" || pokemon.Ability.Name == "illuminate") {
				return []StateEvent{
					SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				}, []string{fmt.Sprintf("%s's accuracy cannot be lowered", pokemon.Name())}
			}

			pokemon.ChangeAccuracy(event.Change)
		case STAT_EVASION:
			pokemon.ChangeEvasion(event.Change)
		}

		absChange := int(math.Abs(float64(event.Change)))
		var message []string = nil

		if event.Change > 0 {
			message = []string{fmt.Sprintf("%s's %s increased by %d stages!", pokemon.Name(), event.StatName, absChange)}
		} else {
			message = []string{fmt.Sprintf("%s's %s decreased by %d stages!", pokemon.Name(), event.StatName, absChange)}
		}

		return nil, message
	} else {
		internalLogger.WithName("stat_change_event").Info("stat change check failed", "stat_check", statCheck, "stat_chance", event.Chance, "pokemon_name", pokemon.Name())
		return nil, nil
	}
}

type AilmentEvent struct {
	PlayerIndex int
	Ailment     int
}

var ailmentApplicationMessages = map[int]string{
	STATUS_NONE:   "%s has been cured of it's afflictions!",
	STATUS_SLEEP:  "%s has fallen asleep!",
	STATUS_PARA:   "%s has been paralyzed!",
	STATUS_FROZEN: "%s has been frozen!",
	STATUS_BURN:   "%s has been burned!",
	STATUS_POISON: "%s has been poisoned!",
	STATUS_TOXIC:  "%s has been badly poisoned!",
}

func (event AilmentEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	// If pokemon already has ailment, return early
	if pokemon.Status != STATUS_NONE {
		return nil, []string{ailmentApplicationMessages[STATUS_NONE]}
	}

	rng := gameState.CreateRng()

	// Pre-Ailment checks
	switch event.Ailment {
	case STATUS_PARA:
		if pokemon.Ability.Name == "limber" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be paralyzed", pokemon.Name()),
			}, nil
		}
	// Set how many turns the pokemon is asleep for
	case STATUS_SLEEP:
		cantSleepMsg := fmt.Sprintf("%s cannot fall asleep", pokemon.Name())
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

		randTime := rng.IntN(2) + 1

		if pokemon.Ability.Name == "early-bird" {
			randTime = int(math.Floor(float64(randTime) / 2.0))
		}

		pokemon.SleepCount = randTime
		internalLogger.WithName("ailment_event").Info("Pokemon fell asleep", "pokemon_name", pokemon.Name(), "sleep_turns", pokemon.SleepCount)
	case STATUS_BURN:
		if pokemon.Ability.Name == "water-veil" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be burned", pokemon.Name()),
			}, nil
		}
	case STATUS_POISON:
		if pokemon.Ability.Name == "immunity" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be poisoned", pokemon.Name()),
			}, nil
		}
	case STATUS_FROZEN:
		if pokemon.Ability.Name == "magma-armor" || gameState.Weather == WEATHER_SUN || gameState.Weather == WEATHER_EXTREME_SUN {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be frozen", pokemon.Name()),
			}, nil
		}
	case STATUS_TOXIC:
		// Block toxic with immunity
		if pokemon.Ability.Name == "immunity" {
			return []StateEvent{
				SimpleAbilityActivationEvent(gameState, event.PlayerIndex),
				NewFmtMessageEvent("%s cannot be poisoned", pokemon.Name()),
			}, nil
		}

		// otherwise init toxic count
		pokemon.ToxicCount = 1
	}

	events := make([]StateEvent, 0)

	pokemon.Status = event.Ailment
	if pokemon.Ability.Name == "synchronize" {
		events = append(events, SimpleAbilityActivationEvent(gameState, event.PlayerIndex))

		switch pokemon.Status {
		case STATUS_BURN, STATUS_POISON, STATUS_PARA:
			events = append(events, AilmentEvent{PlayerIndex: InvertPlayerIndex(event.PlayerIndex), Ailment: pokemon.Status})
		case STATUS_TOXIC:
			events = append(events, AilmentEvent{PlayerIndex: InvertPlayerIndex(event.PlayerIndex), Ailment: STATUS_POISON})
		}
	}

	return events, []string{fmt.Sprintf(ailmentApplicationMessages[event.Ailment], pokemon.Name())}
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
		messages = append(messages, fmt.Sprintf("%s activated their ability: %s", activatorPkm.Name(), event.AbilityName))
	} else {
		messages = append(messages, event.CustomMessage)
	}

	// NOTE: This assumes that all abilities have met their conditions to be activated.
	switch event.AbilityName {
	case "flash-fire":
		activatorPkm.FlashFire = true

		messages = []string{fmt.Sprintf("%s boosted it's fire-type attacks!", activatorPkm.Name())}
	case "lightning-rod":
		events = append(events, NewStatChangeEvent(event.ActivatorInt, STAT_SPATTACK, 1, 100))
	case "volt-absorb", "water-absorb":
		events = append(events, HealPercEvent{HealPerc: .25, PlayerIndex: event.ActivatorInt})
	case "speed-boost":
		events = append(events, NewStatChangeEvent(event.ActivatorInt, STAT_SPEED, 1, 100))
	case "rain-dish":
		events = append(events, HealPercEvent{HealPerc: 1.0 / 16.0, PlayerIndex: event.ActivatorInt})

		messages = []string{fmt.Sprintf("%s was healed by the rain!", activatorPkm.Name())}
	case "shed-skin":
		// TODO: actually test this ability
		activatorPkm.Status = STATUS_NONE

		messages = []string{fmt.Sprintf("%s shed it's skin!", activatorPkm.Name())}
	case "truant":
		activatorPkm.CanAttackThisTurn = false
		activatorPkm.TruantShouldActivate = false

		messages = []string{fmt.Sprintf("%s is loafing around!", activatorPkm.Name())}
	case "forecast":
		switch gameState.Weather {
		case WEATHER_RAIN:
			activatorPkm.BattleType = &TYPE_WATER
		case WEATHER_SUN:
			activatorPkm.BattleType = &TYPE_FIRE
		default:
			activatorPkm.BattleType = &TYPE_NORMAL
		}

		messages = append(messages, fmt.Sprintf("%s changed type to %s", activatorPkm.Name(), activatorPkm.BattleType.Name))
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
		fmt.Sprintf("%s took %d%% damage!", pokemon.Name(), int(damagePercent)),
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
		fmt.Sprintf("%s healed %d%% of their health!", pokemon.Name(), int(healPerc)),
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
		fmt.Sprintf("%s healed by %d%%!", pokemon.Name(), int(heal)),
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
			fmt.Sprintf("%s is burned!", pokemon.Name()),
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
		fmt.Sprintf("%s is poisoned!", pokemon.Name()),
	}
}

type ToxicEvent struct {
	PlayerIndex int
}

func (event ToxicEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	damage := (pokemon.MaxHp / 16) * uint(pokemon.ToxicCount)
	pokemon.ToxicCount++

	internalLogger.WithName("toxic_event").Info("toxic updated", "damage", damage, "toxic_count", pokemon.ToxicCount, "pokemon_name", pokemon.Name())

	return []StateEvent{DamageEvent{Damage: damage, PlayerIndex: event.PlayerIndex}}, []string{
		fmt.Sprintf("%s is badly poisoned!", pokemon.Name()),
	}
}

type FrozenEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event FrozenEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	rng := gameState.CreateRng()

	thawChance := .20
	thawCheck := rng.Float64()

	message := ""

	// pokemon stays frozen
	if thawCheck > thawChance {
		internalLogger.WithName("frozen_event").Info("Thaw check failed", "thaw_check", thawCheck, "thaw_chance", thawChance, "pokemon_name", pokemon.Name())
		message = fmt.Sprintf("%s is frozen and cannot move", pokemon.Name())

		pokemon.CanAttackThisTurn = false
	} else {
		internalLogger.WithName("frozen_event").Info("thaw check passed!", "thaw_check", thawCheck, "thaw_chance", thawChance, "pokemon_name", pokemon.Name())
		message = fmt.Sprintf("%s thawed out!", pokemon.Name())

		// No need for a new event really
		pokemon.Status = STATUS_NONE
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

	rng := gameState.CreateRng()

	paraChance := 0.5
	paraCheck := rng.Float64()

	messages := make([]string, 0)
	messages = append(messages, fmt.Sprintf("%s is paralyzed.", pokemon.Name()))

	if paraCheck > paraChance {
		// don't get para'd
		internalLogger.WithName("para_event").Info("Para Check passed", "para_check", paraCheck, "para_chance", paraChance, "pokemon_name", pokemon.Name())
		return []StateEvent{event.FollowUpAttackEvent}, messages
	} else {
		// do get para'd
		internalLogger.WithName("para_event").Info("Para Check failed", "para_check", paraCheck, "para_chance", paraChance, "pokemon_name", pokemon.Name())
		pokemon.CanAttackThisTurn = false

		messages = append(messages, fmt.Sprintf("%s is paralyzed and cannot move.", pokemon.Name()))
	}

	return nil, messages
}

type FlinchEvent struct {
	PlayerIndex int
}

func (event FlinchEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.CanAttackThisTurn = false

	return nil, []string{fmt.Sprintf("%s flinched and cannot move!", pokemon.Name())}
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
		pokemon.Status = STATUS_NONE
		message = fmt.Sprintf("%s woke up!", pokemon.Name())
		pokemon.CanAttackThisTurn = true
		pokemon.TruantShouldActivate = false
	} else {
		message = fmt.Sprintf("%s is asleep", pokemon.Name())
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

	rng := gameState.CreateRng()

	// TODO: add message
	if pokemon.Ability.Name != "own-tempo" {
		confusionDuration := rng.IntN(3) + 2
		pokemon.ConfusionCount = confusionDuration

		internalLogger.WithName("apply_confusion_event").Info("confusion applied", "confusion_count", pokemon.ConfusionCount, "pokemon_name", pokemon.Name())
	}

	return nil, []string{fmt.Sprintf("%s is now confused!", pokemon.Name())}
}

type ConfusionEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event ConfusionEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()
	pokemon.ConfusionCount--
	internalLogger.WithName("confusion_event").Info("confusion updated", "confusion_count", pokemon.ConfusionCount, "pokemon_name", pokemon.Name())

	rng := gameState.CreateRng()

	confuseText := fmt.Sprintf("%s is confused", pokemon.Name())

	messages := make([]string, 0)
	messages = append(messages, confuseText)

	confChance := .33
	confCheck := rng.Float64()

	// Exit early
	if confCheck > confChance {
		return []StateEvent{event.FollowUpAttackEvent}, messages
	}

	confMove := Move{
		Name:  "Confusion",
		Power: 40,
		Meta: MoveMeta{
			Category: struct {
				Id   int    `json:"id"`
				Name string `json:"name"`
			}{
				Name: "damage",
			},
		},
		DamageClass: DAMAGETYPE_PHYSICAL,
	}

	pokemon.CanAttackThisTurn = false
	dmg := Damage(*pokemon, *pokemon, confMove, false, WEATHER_NONE, rng)

	messages = append(messages, fmt.Sprintf("%s hit itself in confusion.", pokemon.Name()))

	events := make([]StateEvent, 0)
	events = append(events, DamageEvent{Damage: dmg, PlayerIndex: event.PlayerIndex})

	internalLogger.WithName("confusion_event").Info("pokemon hit itself in confusion", "pokemon_name", pokemon.Name())

	return events, messages
}

type ApplyInfatuationEvent struct {
	PlayerIndex int
	Target      int
}

func (event ApplyInfatuationEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	evPlayer, opPlayer := getPlayerPair(gameState, event.PlayerIndex)
	pokemon := evPlayer.GetActivePokemon()

	pokemon.InfatuationTarget = event.Target

	return nil, []string{fmt.Sprintf("%s is infatuated with %s", pokemon.Name(), opPlayer.Team[event.Target].Name())}
}

type InfatuationEvent struct {
	PlayerIndex         int
	FollowUpAttackEvent StateEvent
}

func (event InfatuationEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	infatuationChance := .50
	infatuationCheck := gameState.CreateRng().Float64()

	if infatuationCheck > infatuationChance {
		messages := make([]string, 0)
		internalLogger.WithName("infat_event").Info("infat pokemon failed attack", "pokemon_name", pokemon.Name(), "infat_check", infatuationCheck)
		messages = append(messages, fmt.Sprintf("%s cannot attack because they are infatuation with the enemy!", pokemon.Name()))
		return nil, messages
	} else {
		internalLogger.WithName("infat_event").Info("infat pokemon attacked", "pokemon_name", pokemon.Name(), "infat_check", infatuationCheck)
		return []StateEvent{event.FollowUpAttackEvent}, nil
	}
}

type SandstormDamageEvent struct {
	PlayerIndex int
}

var (
	sandNonDamageTypes     = []*PokemonType{&TYPE_ROCK, &TYPE_STEEL, &TYPE_GROUND}
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
		fmt.Sprintf("%s was damaged by the sandstorm!", pokemon.Name()),
	}
	dmgInt := uint(math.Ceil(dmg))
	return []StateEvent{
		DamageEvent{Damage: dmgInt, PlayerIndex: event.PlayerIndex, SupressMessage: true},
	}, messages
}

type HailDamageEvent struct {
	PlayerIndex int
}

var (
	hailNonDamageTypes     = []*PokemonType{&TYPE_ICE}
	hailNonDamageAbilities = []string{"ice-body", "snow-cloak", "magic-guard", "overcoat"}
)

func (event HailDamageEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.PlayerIndex).GetActivePokemon()

	if slices.Contains(hailNonDamageTypes, pokemon.Base.Type1) || slices.Contains(sandNonDamageTypes, pokemon.Base.Type2) {
		return nil, nil
	}

	if slices.Contains(hailNonDamageAbilities, pokemon.Ability.Name) {
		return nil, nil
	}

	if pokemon.Item == "safety-goggles" {
		return nil, nil
	}

	dmg := float64(pokemon.MaxHp) * (1.0 / 16.0)
	messages := []string{
		"Hail continues to fall",
		fmt.Sprintf("%s was buffeted by the Hail!", pokemon.Name()),
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
	rng := gameState.CreateRng()

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
		check := rng.Float32()
		if check <= .33 && playerPokemon.Status != STATUS_NONE {
			events = append(events,
				SimpleAbilityActivationEvent(gameState, event.PlayerID),
			)
		}
	}

	return events, nil
}

type TypeChangeEvent struct {
	ChangerInt  int
	PokemonType PokemonType
}

func (event TypeChangeEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	pokemon := gameState.GetPlayer(event.ChangerInt).GetActivePokemon()
	pokemon.BattleType = &event.PokemonType

	return nil, []string{fmt.Sprintf("%s changed its type to %s!", pokemon.Name(), event.PokemonType.Name)}
}

type FinalUpdatesEvent struct{}

func decrementTaunt(pokemon *Pokemon) {
	pokemon.TauntCount = max(0, pokemon.TauntCount-1)
}

func (event FinalUpdatesEvent) Update(gameState *GameState) ([]StateEvent, []string) {
	decrementTaunt(gameState.ClientPlayer.GetActivePokemon())
	decrementTaunt(gameState.HostPlayer.GetActivePokemon())

	messages := make([]string, 0)

	if gameState.DisabledWeather != WEATHER_NONE && !gameState.AbilityInPlay("cloud-nine") && !gameState.AbilityInPlay("air-lock") {
		gameState.Weather = gameState.DisabledWeather
		gameState.DisabledWeather = WEATHER_NONE
		messages = append(messages, "The effects of weather have reappeared")
	}

	return nil, messages
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
	internalLogger.WithName("event_iter").Info("Updating state", "event_name", reflect.TypeOf(headEvent))
	followUpEvents, messages := headEvent.Update(state)

	// pop queue
	iter.events = iter.events[1:len(iter.events)]

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
