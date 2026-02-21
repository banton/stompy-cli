package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	configDirName  = ".stompy"
	configFileName = "config"
	configFileType = "yaml"

	defaultAPIURL       = "https://api.stompy.ai/api/v1"
	defaultOutputFormat = "table"
)

// GetConfigDir returns the path to the stompy config directory (~/.stompy).
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", configDirName)
	}
	return filepath.Join(home, configDirName)
}

// GetConfigPath returns the path to the stompy config file (~/.stompy/config.yaml).
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), configFileName+"."+configFileType)
}

// Load initializes Viper, sets defaults, and reads the config file if it exists.
func Load() error {
	viper.SetDefault("api_url", defaultAPIURL)
	viper.SetDefault("output_format", defaultOutputFormat)

	viper.SetConfigName(configFileName)
	viper.SetConfigType(configFileType)
	viper.AddConfigPath(GetConfigDir())

	viper.SetEnvPrefix("STOMPY")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}
	return nil
}

// Save writes the current Viper config to the config file,
// creating the directory if needed.
func Save() error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	return viper.WriteConfigAs(GetConfigPath())
}

// GetAPIURL returns the configured API URL.
func GetAPIURL() string {
	return viper.GetString("api_url")
}

// GetAPIKey returns the configured API key.
func GetAPIKey() string {
	return viper.GetString("api_key")
}

// GetDefaultProject returns the configured default project.
func GetDefaultProject() string {
	return viper.GetString("default_project")
}

// GetOutputFormat returns the configured output format.
func GetOutputFormat() string {
	return viper.GetString("output_format")
}

// SetValue sets a config key to the given value and saves.
func SetValue(key, value string) error {
	viper.Set(key, value)
	return Save()
}

// GetValue returns the string value for a config key.
func GetValue(key string) string {
	return viper.GetString(key)
}

// GetAllSettings returns all config settings as a map.
func GetAllSettings() map[string]any {
	return viper.AllSettings()
}

// SaveTokens persists auth tokens and user info to the config file.
func SaveTokens(accessToken, refreshToken string, expiry time.Time, email, userID string) error {
	viper.Set("auth.access_token", accessToken)
	viper.Set("auth.refresh_token", refreshToken)
	viper.Set("auth.token_expiry", expiry.Format(time.RFC3339))
	viper.Set("auth.email", email)
	viper.Set("auth.user_id", userID)
	return Save()
}

// GetAccessToken returns the stored access token.
func GetAccessToken() string {
	return viper.GetString("auth.access_token")
}

// GetRefreshToken returns the stored refresh token.
func GetRefreshToken() string {
	return viper.GetString("auth.refresh_token")
}

// GetTokenExpiry returns the stored token expiry time.
func GetTokenExpiry() time.Time {
	s := viper.GetString("auth.token_expiry")
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// GetEmail returns the stored user email.
func GetEmail() string {
	return viper.GetString("auth.email")
}

// ClearTokens removes all auth tokens from the config and saves.
func ClearTokens() error {
	viper.Set("auth.access_token", "")
	viper.Set("auth.refresh_token", "")
	viper.Set("auth.token_expiry", "")
	viper.Set("auth.email", "")
	viper.Set("auth.user_id", "")
	return Save()
}

// ResolveProject determines the active project using this precedence:
// 1. Explicit flag value
// 2. STOMPY_PROJECT environment variable
// 3. default_project from config
// Returns an error if no project can be resolved.
func ResolveProject(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("STOMPY_PROJECT"); env != "" {
		return env, nil
	}
	if dp := GetDefaultProject(); dp != "" {
		return dp, nil
	}
	return "", fmt.Errorf("no project specified. Set a default with:\n  stompy project use <name>\n\nOr pass -p <name> to any command. Run 'stompy project list' to see available projects")
}
