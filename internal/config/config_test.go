package config

import (
	"testing"
)

func TestLoadFlagOverride(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "env-token")
	t.Setenv("LOGFIRE_PROJECT", "env-project")

	cfg, err := Load("flag-token", "flag-project")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Token != "flag-token" {
		t.Errorf("expected flag-token, got %s", cfg.Token)
	}
	if cfg.Project != "flag-project" {
		t.Errorf("expected flag-project, got %s", cfg.Project)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "env-token")
	t.Setenv("LOGFIRE_PROJECT", "env-project")
	t.Setenv("LOGFIRE_BASE_URL", "https://custom.example.com")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Token != "env-token" {
		t.Errorf("expected env-token, got %s", cfg.Token)
	}
	if cfg.Project != "env-project" {
		t.Errorf("expected env-project, got %s", cfg.Project)
	}
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %s", cfg.BaseURL)
	}
}

func TestLoadDefaultBaseURL(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_PROJECT", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.BaseURL != "https://logfire-us.pydantic.dev" {
		t.Errorf("expected default base URL, got %s", cfg.BaseURL)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_PROJECT", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load should not fail on missing config file: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}
