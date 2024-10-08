package game

import (
	"encoding/json"
	"io"
	"os"
)

func LoadAbilities(abilityFile string) (map[string][]string, error) {
	file, err := os.Open(abilityFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	abilityMap := make(map[string][]string)
	err = json.Unmarshal(fileData, &abilityMap)

	return abilityMap, err
}
