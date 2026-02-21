package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the stompy CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stompy-cli version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
