package config

import (
	"os"
	"encoding/json"
	"fmt"
)

const configFileName string = ".gatorconfig.json"

type Config struct {
	DBURL string
	UserName string
}


func Read() (Config, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	cfg := Config{}
	jErr := json.NewDecoder(file).Decode(&cfg)
	if jErr != nil {
		return Config{}, jErr
	}

	return cfg, nil
}


func (cfg Config) SetUser(name string) error {
	cfg.UserName = name
	if err := write(cfg); err != nil {
		return err
	}
	return nil
}


func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", homeDir, configFileName), nil
}


func write(cfg Config) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	wErr := os.WriteFile(filePath, data, 0644)
	if wErr != nil {
		return wErr
	}
	return nil
}
