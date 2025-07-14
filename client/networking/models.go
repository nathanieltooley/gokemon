package networking

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/rs/zerolog/log"
)

const (
	MESSAGE_FORCESWITCH messageType = iota
	MESSAGE_TURNRESOLVE
	MESSAGE_GAMEOVER
	MESSAGE_CONTINUE
	MESSAGE_SENDACTION
	MESSAGE_UPDATETIMER
)

const (
	DIR_SYNC = iota
	DIR_CLIENT_PAUSE
)

// "Messages" are during a game for communication
type (
	ForceSwitchMessage struct {
		ForThisPlayer bool
		Events        EventSlice
	}
	TurnResolvedMessage struct {
		Events EventSlice
	}
	GameOverMessage struct {
		// The "you" in this sense is the player who is receiving the message
		YouLost bool
	}
	ContinueUpdaterMessage struct {
		Actions []stateCore.Action
	}
	SendActionMessage struct {
		Action stateCore.Action
	}
	UpdateTimerMessage struct {
		Directive     int
		NewHostTime   int64
		NewClientTime int64
		HostPaused    bool
		ClientPaused  bool
	}
)

type EventSlice struct {
	Events []stateCore.StateEvent
}

func (es EventSlice) GobEncode() ([]byte, error) {
	encodeBuffer := bytes.Buffer{}
	innerEncoder := gob.NewEncoder(&encodeBuffer)
	log.Debug().Msg("encoding eventslice")

	// store eventslice len ahead of time
	if err := innerEncoder.Encode(len(es.Events)); err != nil {
		return nil, err
	}

	for _, event := range es.Events {
		log.Debug().Msgf("encoding event: %+v", event)
		var err error = nil
		// NOTE:
		// Dear Humble Viewer of this "Code"
		// I really did not want to do this. Alas, my Go inexperience has caught up to me
		// and I honest to god don't know how to do this any other way. Perhaps there was something I
		// missed in std.reflect or some macro-like codegen solution? In any case if you know how to write this in a better way,
		// please do not hesitate to submit a PR and fix this.
		switch event := event.(type) {
		case stateCore.SwitchEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.AttackEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.WeatherEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.StatChangeEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.AilmentEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.AbilityActivationEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.DamageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.HealEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.HealPercEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.BurnEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.PoisonEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.ToxicEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.FrozenEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.SleepEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.ParaEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.FlinchEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.ApplyConfusionEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.ConfusionEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.SandstormDamageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.TurnStartEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case *stateCore.EndOfTurnAbilityCheck:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.MessageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case stateCore.FmtMessageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		default:
			return nil, fmt.Errorf("%s has not been given an encoder case", reflect.TypeOf(event).Name())
		}

		if err != nil {
			return nil, err
		}
	}

	return encodeBuffer.Bytes(), nil
}

func (es *EventSlice) GobDecode(buf []byte) error {
	innerDecoder := gob.NewDecoder(bytes.NewBuffer(buf))

	eventLen := 0
	if err := innerDecoder.Decode(&eventLen); err != nil {
		return err
	}

	newEventSlice := make([]stateCore.StateEvent, eventLen)
	for i := range eventLen {
		eventName := ""
		if err := innerDecoder.Decode(&eventName); err != nil {
			return err
		}

		var ev stateCore.StateEvent = nil
		var err error = nil

		// NOTE:
		// Dear Humble Viewer of this "Code"
		// I really did not want to do this. Alas, my Go inexperience has caught up to me
		// and I honest to god don't know how to do this any other way. Perhaps there was something I
		// missed in std.reflect or some macro-like codegen solution? In any case if you know how to write this in a better way,
		// please do not hesitate to submit a PR and fix this.
		switch eventName {
		case "SwitchEvent":
			ev, err = gobDecodeEvent[stateCore.SwitchEvent](innerDecoder)
		case "AttackEvent":
			ev, err = gobDecodeEvent[stateCore.AttackEvent](innerDecoder)
		case "WeatherEvent":
			ev, err = gobDecodeEvent[stateCore.WeatherEvent](innerDecoder)
		case "StatChangeEvent":
			ev, err = gobDecodeEvent[stateCore.StatChangeEvent](innerDecoder)
		case "AilmentEvent":
			ev, err = gobDecodeEvent[stateCore.AilmentEvent](innerDecoder)
		case "AbilityActivationEvent":
			ev, err = gobDecodeEvent[stateCore.AbilityActivationEvent](innerDecoder)
		case "DamageEvent":
			ev, err = gobDecodeEvent[stateCore.DamageEvent](innerDecoder)
		case "HealEvent":
			ev, err = gobDecodeEvent[stateCore.HealEvent](innerDecoder)
		case "HealPercEvent":
			ev, err = gobDecodeEvent[stateCore.HealPercEvent](innerDecoder)
		case "BurnEvent":
			ev, err = gobDecodeEvent[stateCore.BurnEvent](innerDecoder)
		case "PoisonEvent":
			ev, err = gobDecodeEvent[stateCore.PoisonEvent](innerDecoder)
		case "ToxicEvent":
			ev, err = gobDecodeEvent[stateCore.ToxicEvent](innerDecoder)
		case "FrozenEvent":
			ev, err = gobDecodeEvent[stateCore.FrozenEvent](innerDecoder)
		case "SleepEvent":
			ev, err = gobDecodeEvent[stateCore.SleepEvent](innerDecoder)
		case "ParaEvent":
			ev, err = gobDecodeEvent[stateCore.ParaEvent](innerDecoder)
		case "FlinchEvent":
			ev, err = gobDecodeEvent[stateCore.FlinchEvent](innerDecoder)
		case "ApplyConfusionEvent":
			ev, err = gobDecodeEvent[stateCore.ApplyConfusionEvent](innerDecoder)
		case "ConfusionEvent":
			ev, err = gobDecodeEvent[stateCore.ConfusionEvent](innerDecoder)
		case "SandstormDamageEvent":
			ev, err = gobDecodeEvent[stateCore.SandstormDamageEvent](innerDecoder)
		case "TurnStartEvent":
			ev, err = gobDecodeEvent[stateCore.TurnStartEvent](innerDecoder)
		case "EndOfTurnAbilityCheck":
			ev, err = gobDecodeEvent[stateCore.EndOfTurnAbilityCheck](innerDecoder)
		case "MessageEvent":
			ev, err = gobDecodeEvent[stateCore.MessageEvent](innerDecoder)
		case "FmtMessageEvent":
			ev, err = gobDecodeEvent[stateCore.FmtMessageEvent](innerDecoder)
		default:
			return fmt.Errorf("%s has not been given a decoder case", eventName)
		}

		if err != nil {
			return err
		}

		newEventSlice[i] = ev
	}

	es.Events = newEventSlice
	return nil
}

func gobDecodeEvent[T stateCore.StateEvent](decoder *gob.Decoder) (T, error) {
	ev := new(T)
	if err := decoder.Decode(ev); err != nil {
		return *ev, err
	}

	return *ev, nil
}

func gobEncodeEvent[T stateCore.StateEvent](ev T, encoder *gob.Encoder) error {
	// type names as returned by reflect include package name (i.e. core.SwitchEvent)
	// so split it and only get the type name
	pureTypeName := strings.SplitN(reflect.TypeOf(ev).String(), ".", 2)[1]
	if err := encoder.Encode(pureTypeName); err != nil {
		return err
	}

	if err := encoder.Encode(ev); err != nil {
		return err
	}

	return nil
}

type TeamSelectionPacket struct {
	Team []core.Pokemon
}

type StarterSelectionPacket struct {
	StartingIndex int
}

type NetworkingErrorMsg struct {
	Err    error
	Reason string
}

func (e NetworkingErrorMsg) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = "error occured while networking"
	}
	return fmt.Sprintf("%s: %s", reason, e.Err)
}

type NetReaderInfo struct {
	ActionChan  chan stateCore.Action
	TimerChan   chan UpdateTimerMessage
	MessageChan chan tea.Msg

	Conn net.Conn
}

func (g NetReaderInfo) CloseChans() { // Doesn't take pointers because channels should be pointer types themselves
	close(g.ActionChan)
	close(g.MessageChan)
	close(g.TimerChan)
}
