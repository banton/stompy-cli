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
	"github.com/banton/stompy-cli/internal/update"
	"github.com/spf13/cobra"
)

var (
	flagAPIURL  string
	flagAPIKey  string
	flagProject string
	flagOutput  string
	flagVerbose bool

	apiClient        *api.Client
	updateAvailable  = make(chan string, 1)
)

var rootCmd = &cobra.Command{
	Use:           "stompy",
	Short:         "Stompy CLI — manage projects, contexts, and tickets",
	Long:          `A command-line interface for the Stompy API. Manage projects, contexts, and tickets from your terminal.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Fire off async version check (non-blocking, result printed in PostRun)
		go func() {
			if latest := update.CheckForUpdate(Version, config.GetConfigDir()); latest != "" {
				updateAvailable <- latest
			}
			close(updateAvailable)
		}()

		// Skip auth setup for commands that don't need it.
		// Use full command path to avoid matching subcommands with the same name
		// (e.g. "stompy update" vs "stompy context update").
		cmdPath := cmd.CommandPath()
		switch cmdPath {
		case "stompy login", "stompy logout", "stompy version", "stompy update":
			return config.Load()
		}
		switch cmd.Name() {
		case "completion", "bash", "zsh", "fish", "powershell":
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
	err := rootCmd.Execute()

	// Print update notice (if available) after command output
	select {
	case latest := <-updateAvailable:
		if latest != "" && isTableOutput() {
			fmt.Fprintf(os.Stderr, "\n%s A new version of stompy is available (%s). Run %s to upgrade.\n",
				output.Dim("→"),
				output.Teal(latest),
				output.Teal("stompy update"))
		}
	default:
		// Check didn't complete in time — skip silently
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, output.Error("Error:")+"\n  "+err.Error())
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
	return output.NewFormatter(getOutputFormat())
}

// getOutputFormat returns the resolved output format string.
func getOutputFormat() string {
	if flagOutput != "" {
		return flagOutput
	}
	return config.GetOutputFormat()
}

// isTableOutput returns true when output format is table (colors allowed).
func isTableOutput() bool {
	f := getOutputFormat()
	return f == "" || f == "table"
}
