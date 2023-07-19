package sops

import (
	"github.com/spf13/viper"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/hcvault"
)

func init() {
	Configs["hashicorp-vault"] = &HCVaultConfig{}
}

type HCVaultConfig struct{}

func (c *HCVaultConfig) IsActivated() bool {
	return viper.InConfig("encryption.sops.hc_vault.uris")
}

func (c *HCVaultConfig) KeyGroup() (sops.KeyGroup, error) {
	uris := viper.GetString("encryption.sops.hc_vault.uris")

	hcVaultKeys, err := hcvault.NewMasterKeysFromURIs(uris)
	if err != nil {
		return nil, err
	}

	var keyGroup sops.KeyGroup

	for _, k := range hcVaultKeys {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}

func init() {
	viper.BindEnv("encryption.sops.hc_vault.uris", "TF_BACKEND_HTTP_SOPS_HC_VAULT_URIS")
}
