package teamfs

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/nathanieltooley/gokemon/client/game"
)

var ErrNoSuchTeam = errors.New("no such team exists")

var teamsFileName string = "teams.json"

type SavedTeams map[string][]game.Pokemon

func SaveTeam(filePath string, name string, pokemon []game.Pokemon) error {
	teams, err := LoadTeamMap(filePath)
	if err != nil {
		return err
	}

	serializablePokemon := make([]game.Pokemon, 0)
	for _, pokemon := range pokemon {
		if !pokemon.IsNil() {
			serializablePokemon = append(serializablePokemon, pokemon)
		}
	}

	teams[name] = serializablePokemon

	teamsFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer teamsFile.Close()

	teamsJson, err := json.Marshal(teams)
	if err != nil {
		return err
	}

	if _, err := teamsFile.Write(teamsJson); err != nil {
		return err
	}

	return nil
}

func LoadTeam(filePath string, name string) ([6]*game.Pokemon, error) {
	var team [6]*game.Pokemon

	teams, err := LoadTeamMap(filePath)
	if err != nil {
		return team, err
	}

	mapTeam, ok := teams[name]
	if !ok {
		return team, ErrNoSuchTeam
	}

	for i, pokemon := range mapTeam {
		// This should only happen if a user manually edits the teams.json
		if i > 6 {
			break
		}
		team[i] = &pokemon
	}

	return team, nil
}

func LoadTeamMap(filePath string) (SavedTeams, error) {
	teamFile, err := os.Open(filePath)
	// If there is an error, assume the file doesn't exist
	if err != nil {
		if err := os.MkdirAll(filepath.Dir(filePath), 0666); err != nil {
			return nil, err
		}

		teamFile, err = os.Create(filePath)
		// If we still have errors, then bail
		if err != nil {
			return nil, err
		}
	}
	defer teamFile.Close()

	teamFileBytes, err := io.ReadAll(teamFile)
	if err != nil {
		return nil, err
	}

	teams := make(SavedTeams)
	if err := json.Unmarshal(teamFileBytes, &teams); err != nil {
		// If there is an err, ignore it for now and just continue as if it was empty
		teams = make(SavedTeams)
	}

	return teams, nil
}

func TeamSliceToArray(slice []game.Pokemon) [6]*game.Pokemon {
	var team [6]*game.Pokemon

	for i, pokemon := range slice {
		team[i] = &pokemon
	}

	return team
}
