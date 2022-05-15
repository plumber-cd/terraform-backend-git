package crypt

type EncryptionProvider interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

var EncryptionProviders = make(map[string]EncryptionProvider)
