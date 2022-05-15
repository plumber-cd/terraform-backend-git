package crypt

import (
	"crypto/md5"
	"encoding/hex"
)

// MD5 returns an md5 hash for a given string
func MD5(key string) (string, error) {
	hasher := md5.New()
	if _, err := hasher.Write([]byte(key)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
