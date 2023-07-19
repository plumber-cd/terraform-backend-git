package crypt

import (
	"fmt"
	"log"
	"strconv"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/keyservice"
	sopsjson "go.mozilla.org/sops/v3/stores/json"
	"go.mozilla.org/sops/v3/version"

	sc "github.com/plumber-cd/terraform-backend-git/crypt/sops"
	"github.com/spf13/viper"
)

func init() {
	EncryptionProviders["sops"] = &SOPSEncryptionProvider{}
}

type SOPSEncryptionProvider struct{}

// Encrypt will encrypt the data in buffer and return encrypted result.
func (p *SOPSEncryptionProvider) Encrypt(data []byte) ([]byte, error) {
	keyGroups, err := sc.GetActivatedKeyGroups()
	if err != nil {
		return nil, err
	}

	inputStore := &sopsjson.Store{}
	branches, err := inputStore.LoadPlainFile(data)
	if err != nil {
		return nil, err
	}

	tree := sops.Tree{
		Branches: branches,
		Metadata: sops.Metadata{
			KeyGroups: keyGroups,
			Version:   version.Version,
		},
	}

	if shamirThreshold := viper.GetString("sops.shamir_threshold"); shamirThreshold != "" {
		st, err := strconv.Atoi(shamirThreshold)
		if err != nil {
			return nil, err
		}
		tree.Metadata.ShamirThreshold = st
	}

	dataKey, errs := tree.GenerateDataKeyWithKeyServices([]keyservice.KeyServiceClient{keyservice.NewLocalClient()})
	if len(errs) > 0 {
		return nil, fmt.Errorf("Could not generate data key: %s", errs)
	}

	if err := common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  aes.NewCipher(),
	}); err != nil {
		return nil, err
	}

	outputStore := &sopsjson.Store{}
	return outputStore.EmitEncryptedFile(tree)
}

// Decrypt will decrypt the data in buffer.
func (p *SOPSEncryptionProvider) Decrypt(data []byte) ([]byte, error) {
	inputStore := &sopsjson.Store{}
	tree, err := inputStore.LoadEncryptedFile(data)
	if err != nil {
		return nil, err
	}

	if tree.Metadata.Version == "" {
		log.Println("SOPS metadata version was not set, assuming state was not previously encrypted and returning as-is document")
		return data, nil
	}

	_, err = common.DecryptTree(common.DecryptTreeOpts{
		Cipher:      aes.NewCipher(),
		Tree:        &tree,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})
	if err != nil {
		return nil, err
	}

	outputStore := &sopsjson.Store{}
	return outputStore.EmitPlainFile(tree.Branches)
}

func init() {
	viper.BindEnv("sops.shamir_threshold", "TF_BACKEND_HTTP_SOPS_SHAMIR_THRESHOLD")
}
