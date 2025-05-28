package state

import (
	"slices"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game/core"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
)

func TestSnapClean(t *testing.T) {
	snaps := []stateCore.StateSnapshot{
		{
			State: NewState(make([]core.Pokemon, 0), make([]core.Pokemon, 0)),
		},
		stateCore.NewEmptyStateSnapshot(),
		stateCore.NewMessageOnlySnapshot("Hello World!"),
	}

	newSnaps := cleanStateSnapshots(snaps)

	if len(newSnaps) != 1 {
		t.Fatalf("Incorrect snap size. Wanted 1, got %d. Snaps: %+v", len(newSnaps), newSnaps)
	}

	if !slices.Equal(newSnaps[0].Messages, []string{"Hello World!"}) {
		t.Fatalf("Incorrect state messages. Got %+v", newSnaps[0].Messages)
	}
}
