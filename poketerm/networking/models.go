package networking

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/golurk"
	"github.com/rs/zerolog/log"
)

const (
	MESSAGE_TURNRESOLVE messageType = iota + 1
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
	TurnResolveMessage struct {
		Result golurk.TurnResult
	}
	ContinueUpdaterMessage struct {
		Actions []golurk.Action
	}
	SendActionMessage struct {
		Action golurk.Action
	}
	UpdateTimerMessage struct {
		Directive     int
		NewHostTime   int64
		NewClientTime int64
		HostPaused    bool
		ClientPaused  bool
	}
)

// HACK: This has to be updated when/if golurk.TurnResult is updated
type innerResolve struct {
	Kind          int
	ForThisPlayer bool
	Events        EventSlice
}

func (msg TurnResolveMessage) GobEncode() ([]byte, error) {
	encodeBuffer := bytes.Buffer{}
	innerEncoder := gob.NewEncoder(&encodeBuffer)
	ir := innerResolve{
		Kind:          msg.Result.Kind,
		ForThisPlayer: msg.Result.ForThisPlayer,
		Events:        EventSlice{msg.Result.Events},
	}

	err := innerEncoder.Encode(ir)

	return encodeBuffer.Bytes(), err
}

func (msg *TurnResolveMessage) GobDecode(buf []byte) error {
	innerDecoder := gob.NewDecoder(bytes.NewBuffer(buf))
	ir := innerResolve{}

	err := innerDecoder.Decode(&ir)
	if err != nil {
		return err
	}

	msg.Result.Kind = ir.Kind
	msg.Result.Events = ir.Events.Events
	msg.Result.ForThisPlayer = ir.ForThisPlayer

	return nil
}

type EventSlice struct {
	Events []golurk.StateEvent
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
		// please do not hesitate to submit a PR and fix this. Please. I beg of you.
		switch event := event.(type) {
		case golurk.SwitchEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.AttackEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.WeatherEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.StatChangeEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.AilmentEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.AbilityActivationEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.DamageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.HealEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.HealPercEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.BurnEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.PoisonEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.ToxicEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.FrozenEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.SleepEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.ParaEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.FlinchEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.ApplyConfusionEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.ConfusionEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.InfatuationEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.SandstormDamageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.TurnStartEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.EndOfTurnAbilityCheck:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.TypeChangeEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.FinalUpdatesEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.MessageEvent:
			err = gobEncodeEvent(event, innerEncoder)
		case golurk.FmtMessageEvent:
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

	newEventSlice := make([]golurk.StateEvent, eventLen)
	for i := range eventLen {
		eventName := ""
		if err := innerDecoder.Decode(&eventName); err != nil {
			return err
		}

		var ev golurk.StateEvent = nil
		var err error = nil

		// NOTE:
		// Dear Humble Viewer of this "Code"
		// I really did not want to do this. Alas, my Go inexperience has caught up to me
		// and I honest to god don't know how to do this any other way. Perhaps there was something I
		// missed in std.reflect or some macro-like codegen solution? In any case if you know how to write this in a better way,
		// please do not hesitate to submit a PR and fix this.
		switch eventName {
		case "SwitchEvent":
			ev, err = gobDecodeEvent[golurk.SwitchEvent](innerDecoder)
		case "AttackEvent":
			ev, err = gobDecodeEvent[golurk.AttackEvent](innerDecoder)
		case "WeatherEvent":
			ev, err = gobDecodeEvent[golurk.WeatherEvent](innerDecoder)
		case "StatChangeEvent":
			ev, err = gobDecodeEvent[golurk.StatChangeEvent](innerDecoder)
		case "AilmentEvent":
			ev, err = gobDecodeEvent[golurk.AilmentEvent](innerDecoder)
		case "AbilityActivationEvent":
			ev, err = gobDecodeEvent[golurk.AbilityActivationEvent](innerDecoder)
		case "DamageEvent":
			ev, err = gobDecodeEvent[golurk.DamageEvent](innerDecoder)
		case "HealEvent":
			ev, err = gobDecodeEvent[golurk.HealEvent](innerDecoder)
		case "HealPercEvent":
			ev, err = gobDecodeEvent[golurk.HealPercEvent](innerDecoder)
		case "BurnEvent":
			ev, err = gobDecodeEvent[golurk.BurnEvent](innerDecoder)
		case "PoisonEvent":
			ev, err = gobDecodeEvent[golurk.PoisonEvent](innerDecoder)
		case "ToxicEvent":
			ev, err = gobDecodeEvent[golurk.ToxicEvent](innerDecoder)
		case "FrozenEvent":
			ev, err = gobDecodeEvent[golurk.FrozenEvent](innerDecoder)
		case "SleepEvent":
			ev, err = gobDecodeEvent[golurk.SleepEvent](innerDecoder)
		case "ParaEvent":
			ev, err = gobDecodeEvent[golurk.ParaEvent](innerDecoder)
		case "FlinchEvent":
			ev, err = gobDecodeEvent[golurk.FlinchEvent](innerDecoder)
		case "ApplyConfusionEvent":
			ev, err = gobDecodeEvent[golurk.ApplyConfusionEvent](innerDecoder)
		case "ConfusionEvent":
			ev, err = gobDecodeEvent[golurk.ConfusionEvent](innerDecoder)
		case "InfatuationEvent":
			ev, err = gobDecodeEvent[golurk.InfatuationEvent](innerDecoder)
		case "SandstormDamageEvent":
			ev, err = gobDecodeEvent[golurk.SandstormDamageEvent](innerDecoder)
		case "TurnStartEvent":
			ev, err = gobDecodeEvent[golurk.TurnStartEvent](innerDecoder)
		case "EndOfTurnAbilityCheck":
			ev, err = gobDecodeEvent[golurk.EndOfTurnAbilityCheck](innerDecoder)
		case "FinalUpdatesEvent":
			ev, err = gobDecodeEvent[golurk.FinalUpdatesEvent](innerDecoder)
		case "TypeChangeEvent":
			ev, err = gobDecodeEvent[golurk.TypeChangeEvent](innerDecoder)
		case "MessageEvent":
			ev, err = gobDecodeEvent[golurk.MessageEvent](innerDecoder)
		case "FmtMessageEvent":
			ev, err = gobDecodeEvent[golurk.FmtMessageEvent](innerDecoder)
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

func gobDecodeEvent[T golurk.StateEvent](decoder *gob.Decoder) (T, error) {
	ev := new(T)
	if err := decoder.Decode(ev); err != nil {
		return *ev, err
	}

	return *ev, nil
}

func gobEncodeEvent[T golurk.StateEvent](ev T, encoder *gob.Encoder) error {
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
	Team []golurk.Pokemon
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
	ActionChan  chan golurk.Action
	TimerChan   chan UpdateTimerMessage
	MessageChan chan tea.Msg

	Conn net.Conn
}

func (g NetReaderInfo) CloseChans() { // Doesn't take pointers because channels should be pointer types themselves
	close(g.ActionChan)
	close(g.MessageChan)
	close(g.TimerChan)
}
