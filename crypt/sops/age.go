package sops

import (
	"os"

	"github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/age"
)

func init() {
	Configs["age"] = &AgeConfig{}
}

type AgeConfig struct{}

func (a *AgeConfig) IsActivated() bool {
	_, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_AGE_RECIPIENTS")
	return ok
}

func (a *AgeConfig) KeyGroup() (sops.KeyGroup, error) {
	recepients := os.Getenv("TF_BACKEND_HTTP_SOPS_AGE_RECIPIENTS")

	var keyGroup sops.KeyGroup

	masterKeys, err := age.MasterKeysFromRecipients(recepients)

	if err != nil {
		return nil, err
	}

	for _, k := range masterKeys {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}
