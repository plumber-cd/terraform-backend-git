package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	rootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate docs",
	Long:  `Uses Cobra to generate CLI docs`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		err = doc.GenMarkdownTree(rootCmd, cwd)
		if err != nil {
			log.Fatal(err)
		}
	},
}
