package cmd

import (
	"fmt"
	"strings"

	"github.com/banton/stompy-cli/internal/config"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SetValue(args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("%s = %s\n", args[0], args[1])
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := config.GetValue(args[0])
		if val == "" {
			fmt.Printf("%s is not set\n", args[0])
		} else {
			fmt.Println(val)
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := config.GetAllSettings()
		f := getFormatter()

		var fields []output.KeyValue
		for k, v := range settings {
			val := fmt.Sprintf("%v", v)
			// Mask sensitive values
			if isSensitive(k) && len(val) > 8 {
				val = val[:4] + "..." + val[len(val)-4:]
			}
			fields = append(fields, output.KeyValue{Key: k, Value: val})
		}

		fmt.Print(f.FormatSingle(fields))
		fmt.Printf("\nConfig file: %s\n", config.GetConfigPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret")
}
