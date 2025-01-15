package stateupdater

import (
	"slices"
	"testing"

	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
)

func TestSnapClean(t *testing.T) {
	snaps := []state.StateSnapshot{
		{
			State: state.NewState(make([]game.Pokemon, 0), make([]game.Pokemon, 0)),
		},
		state.NewEmptyStateSnapshot(),
		state.NewMessageOnlySnapshot("Hello World!"),
	}

	newSnaps := cleanStateSnapshots(snaps)

	if len(newSnaps) != 1 {
		t.Fatalf("Incorrect snap size. Wanted 1, got %d. Snaps: %+v", len(newSnaps), newSnaps)
	}

	if !slices.Equal(newSnaps[0].Messages, []string{"Hello World!"}) {
		t.Fatalf("Incorrect state messages. Got %+v", newSnaps[0].Messages)
	}
}
