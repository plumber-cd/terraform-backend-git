package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"github.com/spf13/viper"
)

func init() {
	EncryptionProviders["aes"] = &AESEncryptionProvider{}
}

var (
	ErrEncryptionPassphraseNotSet = errors.New("TF_BACKEND_HTTP_ENCRYPTION_PASSPHRASE was not set")
)

type AESEncryptionProvider struct{}

// getEncryptionPassphrase should check all possible config sources and return a state backend encryption key.
func getEncryptionPassphrase() (string, error) {
	passphrase := viper.GetString("aes.passprase")
	if passphrase == "" {
		return "", ErrEncryptionPassphraseNotSet
	}
	return passphrase, nil
}

// createAesCipher uses this passphrase and creates a cipher from it's md5 hash
func createAesCipher(passphrase string) (cipher.Block, error) {
	key, err := MD5(passphrase)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	return block, nil
}

// createGCM will create new GCM for a given passphrase with the key calculated by createAesCipher.
func createGCM(passphrase string) (cipher.AEAD, error) {
	block, err := createAesCipher(passphrase)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}

// Encrypt will encrypt the data in buffer and return encrypted result.
// For a key it will use md5 hash from the passphrase.
func (p *AESEncryptionProvider) Encrypt(data []byte) ([]byte, error) {
	passphrase, err := getEncryptionPassphrase()
	if err != nil {
		return nil, err
	}

	var ciphertext []byte

	gcm, err := createGCM(passphrase)
	if err != nil {
		return ciphertext, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return ciphertext, err
	}

	ciphertext = gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt will decrypt the data in buffer.
// For a key it will use md5 hash from the passphrase.
func (p *AESEncryptionProvider) Decrypt(data []byte) ([]byte, error) {
	passphrase, err := getEncryptionPassphrase()
	if err != nil {
		if err == ErrEncryptionPassphraseNotSet {
			return data, nil
		}
		return nil, err
	}

	var plaintext []byte

	gcm, err := createGCM(passphrase)
	if err != nil {
		return plaintext, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	result, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		if err.Error() == "cipher: message authentication failed" {
			// Assume it wasn't previously encrypted, return as-is
			return data, nil
		}
		return nil, err
	}
	return result, nil
}
