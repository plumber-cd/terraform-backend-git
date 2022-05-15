package crypt

import (
	"fmt"
	"os"
	"strconv"

	sops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/keyservice"
	sopsjson "go.mozilla.org/sops/v3/stores/json"
	"go.mozilla.org/sops/v3/version"

	sc "github.com/plumber-cd/terraform-backend-git/crypt/sops"
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

	if shamirThreshold, ok := os.LookupEnv("TF_BACKEND_HTTP_SOPS_SHAMIR_THRESHOLD"); ok {
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
	tree, _ := inputStore.LoadEncryptedFile(data)

	if tree.Metadata.Version == "" {
		return data, nil
	}

	_, _ = common.DecryptTree(common.DecryptTreeOpts{
		Cipher:      aes.NewCipher(),
		Tree:        &tree,
		KeyServices: []keyservice.KeyServiceClient{keyservice.NewLocalClient()},
	})

	outputStore := &sopsjson.Store{}
	return outputStore.EmitPlainFile(tree.Branches)
}
