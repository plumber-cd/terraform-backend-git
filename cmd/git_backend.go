package cmd

import (
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/plumber-cd/terraform-backend-git/cmd/discovery"
	"github.com/plumber-cd/terraform-backend-git/server"
)

// gitHTTPBackendConfigPath is a path to the backend tf config to generate
const gitHTTPBackendConfigPath = "git_http_backend.auto.tf"

// gitBackendCmd will generate backend config and then start the wrapper
var gitBackendCmd = &cobra.Command{
	Use:   "git",
	Short: "Start backend in Git storage mode and execute the wrapper",
	Long:  "It will also generate " + gitHTTPBackendConfigPath + " in current working directory pointing to this backend",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		t, err := template.New(gitHTTPBackendConfigPath).Parse(`
terraform {
	backend "http" {
		address = "http://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		lock_address = "http://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		unlock_address = "http://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		username = "{{ .username }}"
		password = "{{ .password }}"
	}
}
		`)
		if err != nil {
			log.Fatal(err)
		}

		username, _ := os.LookupEnv("TF_BACKEND_GIT_HTTP_USERNAME")
		password, _ := os.LookupEnv("TF_BACKEND_GIT_HTTP_PASSWORD")

		addr := strings.Split(viper.GetString("address"), ":")
		p := map[string]string{
			"port":     addr[len(addr)-1],
			"username": username,
			"password": password,
		}

		for _, flag := range []string{"repository", "ref", "state"} {
			if p[flag] = viper.GetString("git." + flag); p[flag] == "" {
				log.Fatalf("%s must be set", flag)
			}
		}

		backendConfig, err := os.OpenFile(gitHTTPBackendConfigPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer backendConfig.Close()

		if err := t.Execute(backendConfig, p); err != nil {
			log.Fatal(err)
		}

		go server.Start()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if err := os.Remove(gitHTTPBackendConfigPath); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	gitBackendCmd.PersistentFlags().StringP("repository", "r", "", "Repository to use as storage")
	viper.BindPFlag("git.repository", gitBackendCmd.PersistentFlags().Lookup("repository"))

	gitBackendCmd.PersistentFlags().StringP("ref", "b", "master", "Ref (branch) to use")
	viper.BindPFlag("git.ref", gitBackendCmd.PersistentFlags().Lookup("ref"))
	viper.SetDefault("git.ref", "master")

	gitBackendCmd.PersistentFlags().StringP("state", "s", "", "Ref (branch) to use")
	viper.BindPFlag("git.state", gitBackendCmd.PersistentFlags().Lookup("state"))

	discovery.RegisterBackend(gitBackendCmd)
}
