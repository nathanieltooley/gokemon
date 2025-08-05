package tests

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/nathanieltooley/gokemon/golurk"
	"github.com/nathanieltooley/gokemon/poketerm/global"
	"github.com/nathanieltooley/gokemon/poketerm/networking"
)

func init() {
	global.StopLogging()
}

func TestEncodeDecodeEvents(t *testing.T) {
	events := []golurk.StateEvent{golurk.SwitchEvent{PlayerIndex: 10, SwitchIndex: 100}, golurk.AttackEvent{AttackerID: 20, MoveID: 30}}
	es := networking.EventSlice{Events: events}

	encodeBuf := bytes.Buffer{}
	encoder := gob.NewEncoder(&encodeBuf)
	if err := encoder.Encode(es); err != nil {
		t.Fatalf("%s", err)
	}

	decoder := gob.NewDecoder(&encodeBuf)
	newEs := networking.EventSlice{}

	if err := decoder.Decode(&newEs); err != nil {
		t.Fatalf("%s", err)
	}

	if !reflect.DeepEqual(es, newEs) {
		t.Fatalf("start and end EventSlice values are not equal: %+v != %+v", es, newEs)
	}
}

func TestEncodeDecodeResolveMessage(t *testing.T) {
	events := []golurk.StateEvent{golurk.SwitchEvent{PlayerIndex: 10, SwitchIndex: 100}, golurk.AttackEvent{AttackerID: 20, MoveID: 30}}
	msg := networking.TurnResolveMessage{Result: golurk.TurnResult{Kind: 3, ForThisPlayer: true, Events: events}}

	encodeBuf := bytes.Buffer{}
	encoder := gob.NewEncoder(&encodeBuf)
	if err := encoder.Encode(msg); err != nil {
		t.Fatalf("%s", err)
	}

	decoder := gob.NewDecoder(&encodeBuf)
	newMsg := networking.TurnResolveMessage{}

	if err := decoder.Decode(&newMsg); err != nil {
		t.Fatalf("%s", err)
	}

	if !reflect.DeepEqual(msg, newMsg) {
		t.Fatalf("start and end EventSlice values are not equal: %+v != %+v", msg, newMsg)
	}
}
