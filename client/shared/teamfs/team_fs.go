package teamfs

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/nathanieltooley/gokemon/client/game"
)

var ErrNoSuchTeam = errors.New("no such team exists")

var teamsFileName string = "teams.json"

type SavedTeams map[string][]game.Pokemon

func SaveTeam(dir string, name string, pokemon []game.Pokemon) error {
	teams, err := LoadTeamMap(dir)
	if err != nil {
		return err
	}

	serializablePokemon := make([]game.Pokemon, 0)
	for _, poke := range pokemon {
		serializablePokemon = append(serializablePokemon, poke)
	}

	teams[name] = serializablePokemon

	savePath := path.Join(dir, teamsFileName)

	teamsFile, err := os.Create(savePath)
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

func LoadTeam(dir string, name string) ([6]*game.Pokemon, error) {
	var team [6]*game.Pokemon

	teams, err := LoadTeamMap(dir)
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

func NewTeamSave(dir string) error {
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	_, err := os.Create(path.Join(dir, teamsFileName))

	return err
}

func LoadTeamMap(dir string) (SavedTeams, error) {
	savePath := path.Join(dir, teamsFileName)

	teamFile, err := os.Open(savePath)
	defer teamFile.Close()
	if err != nil {
		// Create the file if it does not exist
		if errors.Is(err, fs.ErrNotExist) {
			teamFile, err = os.Create(savePath)
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

func TeamSliceToArray(slice []game.Pokemon) [6]*game.Pokemon {
	var team [6]*game.Pokemon

	for i, pokemon := range slice {
		team[i] = &pokemon
	}

	return team
}
