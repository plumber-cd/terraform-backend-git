package sops

import (
	"os"

	sops "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/gcpkms"
)

func init() {
	Configs["gcp-kms"] = &GcpKmsConfig{}
}

type GcpKmsConfig struct{}

func (c *GcpKmsConfig) IsActivated() bool {
	_, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_GCP_KMS_KEYS")
	return ok
}

func (c *GcpKmsConfig) KeyGroup() (sops.KeyGroup, error) {
	keys := os.Getenv("TF_BACKEND_HTTP_SOPS_GCP_KMS_KEYS")

	var keyGroup sops.KeyGroup

	for _, k := range gcpkms.MasterKeysFromResourceIDString(keys) {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}
