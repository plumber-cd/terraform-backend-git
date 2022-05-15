package backend

import (
	"fmt"
	"os"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/plumber-cd/terraform-backend-git/crypt"
)

func getEncryptionProvider() (crypt.EncryptionProvider, error) {
	provider, enabled := os.LookupEnv("TF_BACKEND_HTTP_ENCRYPTION_PROVIDER")
	if enabled {
		if !slices.Contains(maps.Keys(crypt.EncryptionProviders), provider) {
			return nil, fmt.Errorf("Unknown encryption provider %q", provider)
		}
		return crypt.EncryptionProviders[provider], nil
	}

	// For backward compatibility
	_, aesEnabled := os.LookupEnv("TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE")
	if aesEnabled {
		return crypt.EncryptionProviders["aes"], nil
	}

	return nil, nil
}

// encryptIfEnabled if encryption was enabled - return encrypted data, otherwise return the data as-is.
func encryptIfEnabled(state []byte) ([]byte, error) {
	if ep, err := getEncryptionProvider(); err != nil {
		return nil, err
	} else if ep != nil {
		return ep.Encrypt(state)
	}
	return state, nil
}

// decryptIfEnabled if encryption was enabled - return decrypted data, otherwise return the data as-is.
func decryptIfEnabled(state []byte) ([]byte, error) {
	if ep, err := getEncryptionProvider(); err != nil {
		return nil, err
	} else if ep != nil {
		return ep.Decrypt(state)
	}
	return state, nil
}
