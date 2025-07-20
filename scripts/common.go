// Package scripts contains a bunch of scripts to grab data from PokeAPI.
//
// TODO: Perhaps remove these scripts and download / build the sqlite db directly from
// pokeAPI's source code. This would require a lot of changes though.
package scripts

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/nathanieltooley/gokemon/client/game/core"
)

func FollowNamedResource[T any](n core.NamedApiResource) (T, error) {
	response, err := http.Get(n.Url)
	if err != nil {
		// FIX: Feels very hacky
		var t T
		return t, err
	}

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		var t T
		return t, err
	}

	var followedJson T

	if err := json.Unmarshal(bytes, &followedJson); err != nil {
		var t T
		return t, err
	}

	return followedJson, nil
}
