package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFlagOverride(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "env-token")

	cfg, err := Load("flag-token", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Token != "flag-token" {
		t.Errorf("expected flag-token, got %s", cfg.Token)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "env-token")
	t.Setenv("LOGFIRE_BASE_URL", "https://custom.example.com")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Token != "env-token" {
		t.Errorf("expected env-token, got %s", cfg.Token)
	}
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %s", cfg.BaseURL)
	}
}

func TestLoadDefaultBaseURL(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
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
	t.Setenv("LOGFIRE_BASE_URL", "")

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load should not fail on missing config file: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func writeConfigFile(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	configDir = dir
	t.Cleanup(func() { configDir = "" })
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvironmentFromFlag(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"},
			"stage": {"token": "stage-token"}
		}
	}`)

	cfg, err := Load("", "stage")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "stage-token" {
		t.Errorf("expected stage-token, got %s", cfg.Token)
	}
}

func TestLoadEnvironmentFromDefault(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"},
			"stage": {"token": "stage-token"}
		}
	}`)

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "prod-token" {
		t.Errorf("expected prod-token, got %s", cfg.Token)
	}
}

func TestLoadEnvironmentFlagOverridesDefault(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"},
			"stage": {"token": "stage-token"}
		}
	}`)

	cfg, err := Load("", "stage")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "stage-token" {
		t.Errorf("expected stage-token, got %s", cfg.Token)
	}
}

func TestLoadEnvironmentUnknown(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"},
			"stage": {"token": "stage-token"}
		}
	}`)

	_, err := Load("", "dev")
	if err == nil {
		t.Fatal("expected error for unknown environment")
	}
	if want := `unknown environment "dev", available: prod, stage`; err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestLoadEnvironmentRequired(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"environments": {
			"prod": {"token": "prod-token"},
			"stage": {"token": "stage-token"}
		}
	}`)

	_, err := Load("", "")
	if err == nil {
		t.Fatal("expected error when no environment selector")
	}
	if want := "environment is required, available: prod, stage"; err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestLoadEnvironmentWithTokenOverride(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"}
		}
	}`)

	cfg, err := Load("override-token", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "override-token" {
		t.Errorf("expected override-token, got %s", cfg.Token)
	}
}

func TestLoadEnvironmentWithBaseURL(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "custom",
		"environments": {
			"custom": {"token": "custom-token", "base_url": "https://custom.logfire.dev"}
		}
	}`)

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "custom-token" {
		t.Errorf("expected custom-token, got %s", cfg.Token)
	}
	if cfg.BaseURL != "https://custom.logfire.dev" {
		t.Errorf("expected custom base URL, got %s", cfg.BaseURL)
	}
}

func TestLoadEnvironmentBaseURLDefault(t *testing.T) {
	t.Setenv("LOGFIRE_READ_TOKEN", "")
	t.Setenv("LOGFIRE_BASE_URL", "")

	writeConfigFile(t, `{
		"default": "prod",
		"environments": {
			"prod": {"token": "prod-token"}
		}
	}`)

	cfg, err := Load("", "")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.BaseURL != "https://logfire-us.pydantic.dev" {
		t.Errorf("expected default base URL, got %s", cfg.BaseURL)
	}
}
