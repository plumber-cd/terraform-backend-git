package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/plumber-cd/terraform-backend-git/backend"
	"github.com/plumber-cd/terraform-backend-git/server"
	"github.com/plumber-cd/terraform-backend-git/storages/git"
	"github.com/plumber-cd/terraform-backend-git/types"
	"github.com/spf13/cobra"
)

var pidFile = os.TempDir() + "/.terraform-backend-git.pid"

var (
	address    string
	accessLogs bool
)

var rootCmd = &cobra.Command{
	Use:   "terraform-backend-git",
	Short: "Terraform HTTP backend implementation that uses Git as storage",
	// will use known storage types in this repository and start a local HTTP server
	Run: func(cmd *cobra.Command, args []string) {
		if err := writePidFile(); err != nil {
			log.Fatal(err)
		}

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
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the currently running backend",
	Run: func(cmd *cobra.Command, args []string) {
		if piddata, err := ioutil.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(string(piddata)); err == nil {
				if err := syscall.Kill(pid, syscall.SIGTERM); err == nil {
					if err := os.Remove(pidFile); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	},
}

func writePidFile() error {
	if piddata, err := ioutil.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(string(piddata)); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(syscall.Signal(0)); err == nil {
					return fmt.Errorf("pid already running: %d", pid)
				}
			}
		}
	}
	return ioutil.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0664)
}

func main() {
	rootCmd.PersistentFlags().StringVarP(&address, "address", "a", "127.0.0.1:6061", "Specify the listen address")
	rootCmd.PersistentFlags().BoolVarP(&accessLogs, "access-logs", "l", false, "Log HTTP requests to the console")
	rootCmd.AddCommand(stopCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
