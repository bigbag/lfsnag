package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultBaseURL = "https://logfire-us.pydantic.dev"

var configDir string

type fileConfig struct {
	Default      string            `json:"default"`
	Environments map[string]Config `json:"environments"`
}

type Config struct {
	Token   string `json:"token"`
	BaseURL string `json:"base_url"`
	Project string `json:"project"`
}

func Load(flagToken, flagEnv, urlProject string) (*Config, error) {
	cfg := &Config{BaseURL: defaultBaseURL}

	fc, err := loadFile()
	if err != nil {
		return nil, err
	}

	if fc != nil {
		envName := flagEnv
		if envName == "" && urlProject != "" {
			envName, err = envByProject(fc.Environments, urlProject)
			if err != nil {
				return nil, err
			}
		}
		if envName == "" {
			envName = fc.Default
		}
		if envName == "" {
			return nil, fmt.Errorf("environment is required, available: %s", envNames(fc.Environments))
		}

		profile, ok := fc.Environments[envName]
		if !ok {
			return nil, fmt.Errorf("unknown environment %q, available: %s", envName, envNames(fc.Environments))
		}

		cfg.Token = profile.Token
		if profile.BaseURL != "" {
			cfg.BaseURL = profile.BaseURL
		}
		cfg.Project = profile.Project
	}

	if v := os.Getenv("LOGFIRE_READ_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("LOGFIRE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	if flagToken != "" {
		cfg.Token = flagToken
	}

	return cfg, nil
}

func envByProject(envs map[string]Config, project string) (string, error) {
	match := ""
	for name, cfg := range envs {
		if cfg.Project != project {
			continue
		}
		if match != "" {
			return "", fmt.Errorf("ambiguous project %q: profiles %q and %q both match", project, match, name)
		}
		match = name
	}
	return match, nil
}

func configFilePath() (string, error) {
	if configDir != "" {
		return filepath.Join(configDir, "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lfsnag", "config.json"), nil
}

func loadFile() (*fileConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	return &fc, nil
}

func envNames(envs map[string]Config) string {
	names := make([]string, 0, len(envs))
	for k := range envs {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
