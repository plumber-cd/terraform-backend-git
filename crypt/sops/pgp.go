package sops

import (
	"os"

	sops "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/pgp"
)

func init() {
	Configs["pgp"] = &PGPConfig{}
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
