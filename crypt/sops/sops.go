package sops

import (
	"os"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/pgp"
)

func init() {
	Configs["pgp"] = &PGPConfig{}
}

type Config interface {
	IsActivated() bool
	KeyGroup() (sops.KeyGroup, error)
}

var Configs = make(map[string]Config)

func GetActivatedKeyGroups() ([]sops.KeyGroup, error) {
	keyGroups := make([]sops.KeyGroup, 0)

	for _, config := range Configs {
		if config.IsActivated() {
			kg, err := config.KeyGroup()
			if err != nil {
				return nil, err
			}
			keyGroups = append(keyGroups, kg)
		}
	}

	return keyGroups, nil
}

type PGPConfig struct{}

func (c *PGPConfig) IsActivated() bool {
	_, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_PGP_FP")
	return ok
}

func (c *PGPConfig) KeyGroup() (sops.KeyGroup, error) {
	fp := os.Getenv("TF_BACKEND_HTTP_SOPS_PGP_FP")

	var keyGroup sops.KeyGroup

	for _, k := range pgp.MasterKeysFromFingerprintString(fp) {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}
