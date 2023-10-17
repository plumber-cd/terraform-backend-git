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

	_ "github.com/plumber-cd/terraform-backend-git/storages/git" // force it to init
)

// gitHTTPBackendConfigPath is a path to the backend tf config to generate
const gitHTTPBackendConfigPath = "git_http_backend.auto.tf"

// gitBackendCmd will generate backend config and then start the wrapper
var gitBackendCmd = &cobra.Command{
	Use:   "git",
	Short: "Start backend in Git storage mode and execute the wrapper",
	Long:  "It will also generate " + gitHTTPBackendConfigPath + " in current working directory pointing to this backend",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cd := viper.GetString("git.dir")
		if cd != "" {
			if err := os.Chdir(cd); err != nil {
				log.Fatal(err)
			}
		}

		t, err := template.New(gitHTTPBackendConfigPath).Parse(`
terraform {
	backend "http" {
		address = "{{ .protocol }}://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		lock_address = "{{ .protocol }}://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		unlock_address = "{{ .protocol }}://localhost:{{ .port }}/?type=git&repository={{ .repository }}&ref={{ .ref }}&state={{ .state }}"
		skip_cert_verification = {{ .skipHttpsVerification }}
		username = "{{ .username }}"
		password = "{{ .password }}"
	}
}
		`)
		if err != nil {
			log.Fatal(err)
		}

		httpCert := viper.GetString("server.https_cert")
		httpKey := viper.GetString("server.https_key")
		protocol := "http"
		if httpCert != "" && httpKey != "" {
			protocol = "https"
		}

		username := viper.GetString("server.http_username")
		password := viper.GetString("server.http_password")
		skipHttpsVerification := viper.GetString("server.skip_https_verification")
		if skipHttpsVerification == "" {
			skipHttpsVerification = "false"
		}

		addr := strings.Split(viper.GetString("address"), ":")
		p := map[string]string{
			"port":                  addr[len(addr)-1],
			"protocol":              protocol,
			"skipHttpsVerification": skipHttpsVerification,
			"username":              username,
			"password":              password,
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

	gitBackendCmd.PersistentFlags().StringP("dir", "d", "", "Change current working directory")
	viper.BindPFlag("git.dir", gitBackendCmd.PersistentFlags().Lookup("dir"))

	discovery.RegisterBackend(gitBackendCmd)

	viper.BindEnv("server.https_cert", "TF_BACKEND_GIT_HTTPS_CERT")
	viper.BindEnv("server.https_key", "TF_BACKEND_GIT_HTTPS_KEY")
	viper.BindEnv("server.skip_https_verification", "TF_BACKEND_GIT_HTTPS_SKIP_VERIFICATION")
	viper.BindEnv("server.http_username", "TF_BACKEND_GIT_HTTP_USERNAME")
	viper.BindEnv("server.http_password", "TF_BACKEND_GIT_HTTP_PASSWORD")
}
