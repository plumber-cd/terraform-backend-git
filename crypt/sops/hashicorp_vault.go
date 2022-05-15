package sops

import (
	"os"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/hcvault"
)

func init() {
	Configs["hc_vault"] = &HCVaultConfig{}
}

type HCVaultConfig struct{}

func (c *HCVaultConfig) IsActivated() bool {
	_, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_HC_VAULT_URIS")
	return ok
}

func (c *HCVaultConfig) KeyGroup() (sops.KeyGroup, error) {
	uris := os.Getenv("TF_BACKEND_HTTP_SOPS_HC_VAULT_URIS")

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
