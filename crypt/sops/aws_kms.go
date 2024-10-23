package sops

import (
	"strings"

	"github.com/spf13/viper"
	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/kms"
)

func init() {
	Configs["aws-kms"] = &AwsKmsConfig{}
}

type AwsKmsConfig struct{}

func (c *AwsKmsConfig) IsActivated() bool {
	return viper.InConfig("encryption.sops.aws.key_arns")
}

func (c *AwsKmsConfig) KeyGroup() (sops.KeyGroup, error) {
	profile := viper.GetString("encryption.sops.aws.profile")
	arns := viper.GetString("encryption.sops.aws.key_arns")
	contextStr := viper.GetString("encryption.sops.aws.kms_context")
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

func init() {
	viper.BindEnv("encryption.sops.aws.key_arns", "TF_BACKEND_HTTP_SOPS_AWS_KMS_ARNS")
	viper.BindEnv("encryption.sops.aws.profile", "TF_BACKEND_HTTP_SOPS_AWS_PROFILE")
	viper.BindEnv("encryption.sops.aws.kms_context", "TF_BACKEND_HTTP_SOPS_AWS_KMS_CONTEXT")
}
