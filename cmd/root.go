package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/banton/stompy-cli/internal/api"
	"github.com/banton/stompy-cli/internal/auth"
	"github.com/banton/stompy-cli/internal/config"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagAPIURL  string
	flagAPIKey  string
	flagProject string
	flagOutput  string
	flagVerbose bool

	apiClient *api.Client
)

var rootCmd = &cobra.Command{
	Use:   "stompy",
	Short: "Stompy CLI — manage projects, contexts, and tickets",
	Long:  `A command-line interface for the Stompy API. Manage projects, contexts, and tickets from your terminal.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip auth setup for commands that don't need it
		cmdPath := cmd.CommandPath() // e.g. "stompy config set", "stompy ticket get"
		switch cmd.Name() {
		case "login", "logout", "version", "completion", "bash", "zsh", "fish", "powershell":
			return config.Load()
		}
		// Config subcommands don't need API auth
		if strings.Contains(cmdPath, "config ") {
			return config.Load()
		}
		// Also skip for parent commands (just groupings)
		if !cmd.HasParent() || (cmd.HasSubCommands() && len(args) == 0) {
			return config.Load()
		}

		if err := config.Load(); err != nil {
			return err
		}

		token, err := resolveAuthToken()
		if err != nil {
			return err
		}

		apiURL := flagAPIURL
		if apiURL == "" {
			apiURL = config.GetAPIURL()
		}

		apiClient = api.NewClient(apiURL, token, flagVerbose)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url", "", "Override API base URL")
	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "Override API key")
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "Override default project")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Debug HTTP logging")
}

// Execute is the main entry point for the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveAuthToken determines the auth token using precedence:
// --api-key flag > STOMPY_API_KEY env > OAuth token (with auto-refresh) > api_key from config > error
func resolveAuthToken() (string, error) {
	// 1. --api-key flag
	if flagAPIKey != "" {
		return flagAPIKey, nil
	}

	// 2. STOMPY_API_KEY env var
	if envKey := os.Getenv("STOMPY_API_KEY"); envKey != "" {
		return envKey, nil
	}

	// 3. OAuth token from config (with auto-refresh)
	accessToken := config.GetAccessToken()
	if accessToken != "" {
		expiry := config.GetTokenExpiry()
		if !auth.IsExpired(expiry) {
			return accessToken, nil
		}

		// Try to refresh
		refreshToken := config.GetRefreshToken()
		if refreshToken != "" {
			apiURL := flagAPIURL
			if apiURL == "" {
				apiURL = config.GetAPIURL()
			}
			tokenResp, err := auth.RefreshToken(apiURL, refreshToken)
			if err == nil {
				newExpiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
				_ = config.SaveTokens(tokenResp.AccessToken, tokenResp.RefreshToken, newExpiry, config.GetEmail(), "")
				return tokenResp.AccessToken, nil
			}
			// Refresh failed — fall through
		}
	}

	// 4. Static api_key from config
	if apiKey := config.GetAPIKey(); apiKey != "" {
		return apiKey, nil
	}

	// 5. No auth available
	return "", fmt.Errorf("not authenticated. Run 'stompy login' or set STOMPY_API_KEY")
}

// getProject resolves the active project name.
func getProject() (string, error) {
	return config.ResolveProject(flagProject)
}

// getFormatter returns the output formatter based on flags and config.
func getFormatter() output.Formatter {
	format := flagOutput
	if format == "" {
		format = config.GetOutputFormat()
	}
	return output.NewFormatter(format)
}
