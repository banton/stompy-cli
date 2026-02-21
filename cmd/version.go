package cmd

import (
	"fmt"

	"github.com/banton/stompy-cli/internal/config"
	"github.com/banton/stompy-cli/internal/output"
	"github.com/banton/stompy-cli/internal/update"
	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the stompy CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stompy-cli version %s\n", Version)

		// Check for newer version
		if latest := update.CheckForUpdate(Version, config.GetConfigDir()); latest != "" {
			fmt.Printf("%s available: %s (run %s to upgrade)\n",
				output.Dim("Update"),
				output.Teal(latest),
				output.Teal("stompy update"))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
