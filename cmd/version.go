package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version holds the version binary built with - must be injected from build process via -ldflags="-X 'github.com/plumber-cd/terraform-backend-git/cmd.Version==${{ env.RELEASE_VERSION }}'"
var Version = "dev"

// versionCmd will print the version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
