package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/handlers"
	"github.com/plumber-cd/terraform-backend-git/backend"
	"github.com/plumber-cd/terraform-backend-git/server"
	"github.com/plumber-cd/terraform-backend-git/storages/git"
	"github.com/plumber-cd/terraform-backend-git/types"
	"github.com/spf13/cobra"
)

var (
	address    string
	accessLogs bool
)

// rootCmd main command that just starts the server and keeps listening on port until terminated
var rootCmd = &cobra.Command{
	Use:   "terraform-backend-git",
	Short: "Terraform HTTP backend implementation that uses Git as storage",
	// will use known storage types in this repository and start a local HTTP server
	Run: func(cmd *cobra.Command, args []string) {
		if err := lockPidFile(); err != nil {
			log.Fatal(err)
		}

		startServer()
	},
}

// stopCmd will stop the server started via rootCmd via it's pid file
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the currently running backend",
	Run: func(cmd *cobra.Command, args []string) {
		if err := stopPidFile(); err != nil {
			log.Fatal(err)
		}
	},
}

// backendsCmds is a list of backend types available via cmd wrapper
var backendsCmds = []*cobra.Command{
	gitBackendCmd,
}

// wrappersCmds is a list of wrapper commands available to run wrapped into a backend wrapper
// i.e. backand wrapper "git" will start an http backend in Git storage mode
// and "terraform" wrapper started from it will use terraform while that backend is running
var wrappersCmds = []*cobra.Command{
	terraformWrapperCmd,
}

// startServer listen for traffic
func startServer() {
	backend.KnownStorageTypes = map[string]types.StorageClient{
		"git": git.NewStorageClient(),
	}

	http.HandleFunc("/", server.HandleFunc)

	var handler http.Handler
	if accessLogs {
		handler = handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)
	} else {
		handler = nil
	}

	log.Println("listen on", address)
	log.Fatal(http.ListenAndServe(address, handler))
}

func main() {
	// keep the output clean as in wrapper mode it'll mess out with Terraform own output
	log.SetFlags(0)
	log.SetPrefix("[terraform-backend-git]: ")

	rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "127.0.0.1:6061", "Specify the listen address")
	rootCmd.PersistentFlags().BoolVarP(&accessLogs, "access-logs", "l", false, "Log HTTP requests to the console")

	rootCmd.AddCommand(stopCmd)

	gitBackendCmd.PersistentFlags().StringP("repository", "r", "", "Repository to use as storage")
	gitBackendCmd.MarkPersistentFlagRequired("repository")
	gitBackendCmd.PersistentFlags().StringP("ref", "b", "master", "Ref (branch) to use")
	gitBackendCmd.PersistentFlags().StringP("state", "s", "", "Ref (branch) to use")
	gitBackendCmd.MarkPersistentFlagRequired("state")

	terraformWrapperCmd.Flags().StringP("tf", "t", "terraform", "Path to terraform binary")

	// for every backend type CMD add a wrapper CMD behind
	for _, backendCmd := range backendsCmds {
		for _, wrapperCmd := range wrappersCmds {
			wrapperCmd.Flags().SetInterspersed(false)
			backendCmd.AddCommand(wrapperCmd)
		}

		rootCmd.AddCommand(backendCmd)
	}

	if err := rootCmd.Execute(); err != nil {
		// If the error was coming from a wrapper, must respect it's exit code
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			os.Exit(exitErr.ExitCode())
		}

		fmt.Println(err)
		os.Exit(1)
	}
}
