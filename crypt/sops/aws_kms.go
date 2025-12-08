package sops

import (
	"os"
	"strings"

	sops "github.com/getsops/sops/v3"
	"github.com/getsops/sops/v3/kms"
)

func init() {
	Configs["aws-kms"] = &AwsKmsConfig{}
}

type AwsKmsConfig struct{}

func (c *AwsKmsConfig) IsActivated() bool {
	_, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_AWS_KMS_ARNS")
	return ok
}

func (c *AwsKmsConfig) KeyGroup() (sops.KeyGroup, error) {
	profile := os.Getenv("TF_BACKEND_HTTP_SOPS_AWS_PROFILE")
	arns := os.Getenv("TF_BACKEND_HTTP_SOPS_AWS_KMS_ARNS")
	contextStr := os.Getenv("TF_BACKEND_HTTP_SOPS_AWS_KMS_CONTEXT")
	contextStr = strings.TrimSpace(contextStr)

	context := make(map[string]*string)
	for _, pair := range strings.Split(contextStr, ",") {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		context[kv[0]] = &kv[1]
	}

	var keyGroup sops.KeyGroup

	for _, k := range kms.MasterKeysFromArnString(arns, context, profile) {
		keyGroup = append(keyGroup, k)
	}

	return keyGroup, nil
}
