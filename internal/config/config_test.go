package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// setupTestConfig creates a temp dir, points viper at it, and returns a cleanup func.
func setupTestConfig(t *testing.T) string {
	t.Helper()
	viper.Reset()

	tmpDir := t.TempDir()
	// Override home so GetConfigDir uses our temp dir
	t.Setenv("HOME", tmpDir)

	return tmpDir
}

func TestLoadDefaults(t *testing.T) {
	setupTestConfig(t)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got := GetAPIURL(); got != defaultAPIURL {
		t.Errorf("GetAPIURL() = %q, want %q", got, defaultAPIURL)
	}
	if got := GetOutputFormat(); got != defaultOutputFormat {
		t.Errorf("GetOutputFormat() = %q, want %q", got, defaultOutputFormat)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := setupTestConfig(t)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	viper.Set("api_key", "test-key-123")
	viper.Set("default_project", "my-project")

	if err := Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file was written
	configPath := filepath.Join(tmpDir, configDirName, configFileName+"."+configFileType)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", configPath)
	}

	// Reset and re-load
	viper.Reset()
	if err := Load(); err != nil {
		t.Fatalf("Load() after save error: %v", err)
	}

	if got := GetAPIKey(); got != "test-key-123" {
		t.Errorf("GetAPIKey() = %q, want %q", got, "test-key-123")
	}
	if got := GetDefaultProject(); got != "my-project" {
		t.Errorf("GetDefaultProject() = %q, want %q", got, "my-project")
	}
}

func TestSetValueAndGetValue(t *testing.T) {
	setupTestConfig(t)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if err := SetValue("api_key", "new-key"); err != nil {
		t.Fatalf("SetValue() error: %v", err)
	}

	if got := GetValue("api_key"); got != "new-key" {
		t.Errorf("GetValue(api_key) = %q, want %q", got, "new-key")
	}
}

func TestGetAllSettings(t *testing.T) {
	setupTestConfig(t)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	settings := GetAllSettings()
	if _, ok := settings["api_url"]; !ok {
		t.Error("GetAllSettings() missing api_url key")
	}
	if _, ok := settings["output_format"]; !ok {
		t.Error("GetAllSettings() missing output_format key")
	}
}

func TestSaveAndClearTokens(t *testing.T) {
	setupTestConfig(t)

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	fixedTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	if err := SaveTokens("access-tok", "refresh-tok", fixedTime, "user@test.com", "user-123"); err != nil {
		t.Fatalf("SaveTokens() error: %v", err)
	}

	if got := GetAccessToken(); got != "access-tok" {
		t.Errorf("GetAccessToken() = %q, want %q", got, "access-tok")
	}
	if got := GetRefreshToken(); got != "refresh-tok" {
		t.Errorf("GetRefreshToken() = %q, want %q", got, "refresh-tok")
	}
	if got := GetTokenExpiry(); !got.Equal(fixedTime) {
		t.Errorf("GetTokenExpiry() = %v, want %v", got, fixedTime)
	}
	if got := GetEmail(); got != "user@test.com" {
		t.Errorf("GetEmail() = %q, want %q", got, "user@test.com")
	}

	if err := ClearTokens(); err != nil {
		t.Fatalf("ClearTokens() error: %v", err)
	}

	if got := GetAccessToken(); got != "" {
		t.Errorf("after ClearTokens(), GetAccessToken() = %q, want empty", got)
	}
	if got := GetRefreshToken(); got != "" {
		t.Errorf("after ClearTokens(), GetRefreshToken() = %q, want empty", got)
	}
	if got := GetTokenExpiry(); !got.IsZero() {
		t.Errorf("after ClearTokens(), GetTokenExpiry() = %v, want zero", got)
	}
}

func TestResolveProject_FlagValue(t *testing.T) {
	setupTestConfig(t)
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	got, err := ResolveProject("flag-project")
	if err != nil {
		t.Fatalf("ResolveProject() error: %v", err)
	}
	if got != "flag-project" {
		t.Errorf("ResolveProject() = %q, want %q", got, "flag-project")
	}
}

func TestResolveProject_EnvVar(t *testing.T) {
	setupTestConfig(t)
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	t.Setenv("STOMPY_PROJECT", "env-project")

	got, err := ResolveProject("")
	if err != nil {
		t.Fatalf("ResolveProject() error: %v", err)
	}
	if got != "env-project" {
		t.Errorf("ResolveProject() = %q, want %q", got, "env-project")
	}
}

func TestResolveProject_DefaultConfig(t *testing.T) {
	setupTestConfig(t)
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	viper.Set("default_project", "config-project")

	got, err := ResolveProject("")
	if err != nil {
		t.Fatalf("ResolveProject() error: %v", err)
	}
	if got != "config-project" {
		t.Errorf("ResolveProject() = %q, want %q", got, "config-project")
	}
}

func TestResolveProject_Error(t *testing.T) {
	setupTestConfig(t)
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	_, err := ResolveProject("")
	if err == nil {
		t.Error("ResolveProject() expected error when no project available, got nil")
	}
}

func TestResolveProject_Precedence(t *testing.T) {
	setupTestConfig(t)
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Set all three sources
	viper.Set("default_project", "config-project")
	t.Setenv("STOMPY_PROJECT", "env-project")

	// Flag takes precedence
	got, err := ResolveProject("flag-project")
	if err != nil {
		t.Fatalf("ResolveProject() error: %v", err)
	}
	if got != "flag-project" {
		t.Errorf("ResolveProject() = %q, want %q (flag should win)", got, "flag-project")
	}

	// Without flag, env takes precedence over config
	got, err = ResolveProject("")
	if err != nil {
		t.Fatalf("ResolveProject() error: %v", err)
	}
	if got != "env-project" {
		t.Errorf("ResolveProject() = %q, want %q (env should win over config)", got, "env-project")
	}
}
