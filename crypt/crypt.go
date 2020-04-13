package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// MD5 returns an md5 hash for a given string
func MD5(key string) (string, error) {
	hasher := md5.New()
	if _, err := hasher.Write([]byte(key)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
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

// EncryptAES will encrypt the data in buffer and return encrypted result.
// For a key it will use md5 hash from the passphrase provided.
func EncryptAES(data []byte, passphrase string) ([]byte, error) {
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

// DecryptAES will decrypt the data in buffer.
// For a key it will use md5 hash from the passphrase provided.
func DecryptAES(data []byte, passphrase string) ([]byte, error) {
	var plaintext []byte

	gcm, err := createGCM(passphrase)
	if err != nil {
		return plaintext, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}
