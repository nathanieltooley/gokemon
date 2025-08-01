package global

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func DefaultConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "gokemon")
}

func DefaultConfigLocation() string {
	return filepath.Join(DefaultConfigDir(), "config.json")
}

func SaveConfig(config GlobalConfig) error {
	jsonString, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(DefaultConfigLocation(), jsonString, 0777); err != nil {
		return err
	}

	return nil
}
