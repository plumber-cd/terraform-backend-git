package backend

import (
	"os"

	"github.com/plumber-cd/terraform-backend-git/crypt"
)

// getEncryptionPassphrase should check all possible config sources and return a state backend encryption key.
func getEncryptionPassphrase() string {
	passphrase, _ := os.LookupEnv("TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE")
	return passphrase
}

// encryptIfEnabled if encryption was enabled - return encrypted data, otherwise return the data as-is.
func encryptIfEnabled(state []byte) ([]byte, error) {
	passphrase := getEncryptionPassphrase()

	if passphrase == "" {
		return state, nil
	}

	return crypt.EncryptAES(state, getEncryptionPassphrase())
}

// decryptIfEnabled if encryption was enabled - attempt to decrypt the data. Otherwise return it as-is.
// If decryption fails, it will assume encryption was not enabled previously for this state and return it as-is too.
func decryptIfEnabled(state []byte) ([]byte, error) {
	passphrase := getEncryptionPassphrase()

	if passphrase == "" {
		return state, nil
	}

	buf, err := crypt.DecryptAES(state, getEncryptionPassphrase())
	if err != nil && err.Error() == "cipher: message authentication failed" {
		// Assumei t wasn't previously encrypted, return as-is
		return state, nil
	}
	return buf, err
}
