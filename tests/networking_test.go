package tests

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
	"github.com/nathanieltooley/gokemon/client/networking"
)

func TestEncodeDecodeEvents(t *testing.T) {
	events := []stateCore.StateEvent{stateCore.SwitchEvent{PlayerIndex: 10, SwitchIndex: 100}, stateCore.AttackEvent{AttackerID: 20, MoveID: 30}}
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
