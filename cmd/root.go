package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/plumber-cd/terraform-backend-git/cmd/discovery"
	"github.com/plumber-cd/terraform-backend-git/pid"
	"github.com/plumber-cd/terraform-backend-git/server"
)

var cfgFile string

// rootCmd main command that just starts the server and keeps listening on port until terminated
var rootCmd = &cobra.Command{
	Use:   "terraform-backend-git",
	Short: "Terraform HTTP backend implementation that uses Git as storage",
	// will use known storage types in this repository and start a local HTTP server
	Run: func(cmd *cobra.Command, args []string) {
		if err := pid.LockPidFile(); err != nil {
			log.Fatal(err)
		}

		server.Start()
	},
}

// BackendsCmds is a list of backend types available via cmd wrapper
var BackendsCmds = make([]*cobra.Command, 0)

// WrappersCmds is a list of wrapper commands available to run wrapped into a backend wrapper
// i.e. backand wrapper "git" will start an http backend in Git storage mode
// and "terraform" wrapper started from it will use terraform while that backend is running
var WrappersCmds = make([]*cobra.Command, 0)

func Exec() {
	if err := discovery.Root().Execute(); err != nil {
		// If the error was coming from a wrapper, must respect it's exit code
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			os.Exit(exitErr.ExitCode())
		}

		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// keep the output clean as in wrapper mode it'll mess out with Terraform own output
	log.SetFlags(0)
	log.SetPrefix("[terraform-backend-git]: ")

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is terraform-backend-git.hcl)")

	rootCmd.PersistentFlags().StringP("address", "a", "127.0.0.1:6061", "Specify the listen address")
	viper.BindPFlag("address", rootCmd.PersistentFlags().Lookup("address"))
	viper.SetDefault("address", "127.0.0.1:6061")
	rootCmd.PersistentFlags().BoolP("access-logs", "l", false, "Log HTTP requests to the console")
	viper.BindPFlag("accessLogs", rootCmd.PersistentFlags().Lookup("access-logs"))
	viper.SetDefault("accessLogs", false)

	discovery.RegisterRoot(rootCmd)
}

func initConfig() {
	viper.SetConfigType("hcl")
	viper.SetConfigName("terraform-backend-git")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}
		viper.AddConfigPath(home)

		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		viper.AddConfigPath(cwd)
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TF_BACKEND_GIT")

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
