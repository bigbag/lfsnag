package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const defaultBaseURL = "https://logfire-us.pydantic.dev"

type Config struct {
	Token   string `json:"token"`
	Project string `json:"project"`
	BaseURL string `json:"base_url"`
}

func Load(flagToken, flagProject string) (*Config, error) {
	cfg := &Config{BaseURL: defaultBaseURL}

	if err := cfg.loadFile(); err != nil {
		return nil, err
	}

	if v := os.Getenv("LOGFIRE_READ_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("LOGFIRE_PROJECT"); v != "" {
		cfg.Project = v
	}
	if v := os.Getenv("LOGFIRE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	if flagToken != "" {
		cfg.Token = flagToken
	}
	if flagProject != "" {
		cfg.Project = flagProject
	}

	return cfg, nil
}

func (c *Config) loadFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".config", "lfsnag", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return json.Unmarshal(data, c)
}
