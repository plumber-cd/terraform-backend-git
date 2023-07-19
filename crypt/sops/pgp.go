package sops

import (
	"github.com/spf13/viper"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/pgp"
)

func init() {
	Configs["pgp"] = &PGPConfig{}
}

type PGPConfig struct{}

func (c *PGPConfig) IsActivated() bool {
	return viper.InConfig("encryption.sops.gpg.key_ids")
}

func (c *PGPConfig) KeyGroup() (sops.KeyGroup, error) {
	fp := viper.GetString("encryption.sops.gpg.key_ids")

	var keyGroup sops.KeyGroup

	for _, k := range pgp.MasterKeysFromFingerprintString(fp) {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}

func init() {
	viper.BindEnv("encryption.sops.gpg.key_ids", "TF_BACKEND_HTTP_SOPS_PGP_FP")
}
