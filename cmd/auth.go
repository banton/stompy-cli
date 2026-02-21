package cmd

import (
	"fmt"
	"time"

	"github.com/banton/stompy-cli/internal/auth"
	"github.com/banton/stompy-cli/internal/config"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth 2.0 browser-based login (PKCE)",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiURL := flagAPIURL
		if apiURL == "" {
			apiURL = config.GetAPIURL()
		}

		tokenResp, err := auth.Login(apiURL)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		if err := config.SaveTokens(tokenResp.AccessToken, tokenResp.RefreshToken, expiry, "", ""); err != nil {
			return fmt.Errorf("saving tokens: %w", err)
		}

		fmt.Println("Login successful! Token saved to", config.GetConfigPath())
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored authentication tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.ClearTokens(); err != nil {
			return fmt.Errorf("clearing tokens: %w", err)
		}
		fmt.Println("Logged out. Tokens cleared.")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Load(); err != nil {
			return err
		}

		f := getFormatter()

		// Check API key first
		if flagAPIKey != "" || config.GetAPIKey() != "" {
			fmt.Print(f.FormatSingle([]output.KeyValue{
				{Key: "Auth Method", Value: "API Key"},
				{Key: "Status", Value: "Authenticated"},
			}))
			return nil
		}

		// Check OAuth tokens
		token := config.GetAccessToken()
		if token == "" {
			fmt.Println("Not authenticated. Run 'stompy login' to authenticate.")
			return nil
		}

		expiry := config.GetTokenExpiry()
		email := config.GetEmail()
		status := "Valid"
		if auth.IsExpired(expiry) {
			status = "Expired (will auto-refresh on next command)"
		}

		fields := []output.KeyValue{
			{Key: "Auth Method", Value: "OAuth 2.0 (PKCE)"},
			{Key: "Status", Value: status},
		}
		if email != "" {
			fields = append(fields, output.KeyValue{Key: "Email", Value: email})
		}
		if !expiry.IsZero() {
			fields = append(fields, output.KeyValue{Key: "Token Expiry", Value: expiry.Local().Format(time.RFC3339)})
		}

		fmt.Print(f.FormatSingle(fields))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
}
