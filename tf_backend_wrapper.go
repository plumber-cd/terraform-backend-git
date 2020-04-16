package main

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// terraformWrapperCmd will pass all arguments to terraform,
// given that it was started from under the wrapper - will be pointing to this backend
var terraformWrapperCmd = &cobra.Command{
	Use:                   "terraform",
	Short:                 "Run terraform while storage is running",
	DisableFlagsInUseLine: true,
	SilenceUsage:          true,
	Args:                  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		tf := viper.GetString("wrapper.tf.bin")

		tfCommand := exec.Command(tf, args...)
		tfCommand.Stdin = os.Stdin
		tfCommand.Stdout = os.Stdout
		tfCommand.Stderr = os.Stderr

		if err := tfCommand.Run(); err != nil {
			return err
		}

		return nil
	},
}
