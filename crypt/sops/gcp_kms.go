package sops

import (
	"github.com/spf13/viper"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/gcpkms"
)

func init() {
	Configs["gcp-kms"] = &GcpKmsConfig{}
}

type GcpKmsConfig struct{}

func (c *GcpKmsConfig) IsActivated() bool {
	return viper.InConfig("encryption.sops.gpg.key")

}

func (c *GcpKmsConfig) KeyGroup() (sops.KeyGroup, error) {
	keys := viper.GetString("encryption.sops.gpg.key")

	var keyGroup sops.KeyGroup

	for _, k := range gcpkms.MasterKeysFromResourceIDString(keys) {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}

func init() {
	viper.BindEnv("encryption.sops.gpg.key", "TF_BACKEND_HTTP_SOPS_PGP_FP")
}
