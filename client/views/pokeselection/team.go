package pokeselection

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/nathanieltooley/gokemon/client/game"
)

const TEAM_SAVE_PATH string = "./saves/teams.json"

var ErrNoSuchTeam = errors.New("no such team exists")

type SavedTeams map[string][]game.Pokemon

func SaveTeam(name string, pokemon []*game.Pokemon) error {
	teams, err := LoadTeamMap()
	if err != nil {
		return err
	}

	serializablePokemon := make([]game.Pokemon, 0)
	for _, pokePointer := range pokemon {
		if pokePointer != nil {
			serializablePokemon = append(serializablePokemon, *pokePointer)
		}
	}

	teams[name] = serializablePokemon

	teamsFile, err := os.Create(TEAM_SAVE_PATH)
	defer teamsFile.Close()
	if err != nil {
		return err
	}

	teamsJson, err := json.Marshal(teams)
	if err != nil {
		return err
	}

	if _, err := teamsFile.Write(teamsJson); err != nil {
		return err
	}

	return nil
}

func LoadTeam(name string) ([6]*game.Pokemon, error) {
	var team [6]*game.Pokemon

	teams, err := LoadTeamMap()
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

func LoadTeamMap() (SavedTeams, error) {
	teamFile, err := os.Open(TEAM_SAVE_PATH)
	defer teamFile.Close()
	if err != nil {
		// Create the file if it does not exist
		if errors.Is(err, fs.ErrNotExist) {
			teamFile, err = os.Create(TEAM_SAVE_PATH)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

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
