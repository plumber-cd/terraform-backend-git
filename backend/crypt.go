package backend

import (
	"os"

	"github.com/plumber-cd/terraform-backend-git/crypt"
)

func getEncryptionPassphrase() string {
	passphrase, _ := os.LookupEnv("TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE")
	return passphrase
}

func encryptIfEnabled(state []byte) ([]byte, error) {
	passphrase := getEncryptionPassphrase()

	if passphrase == "" {
		return state, nil
	}

	return crypt.EncryptAES(state, getEncryptionPassphrase())
}

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
