package git

import (
	"errors"
	"os"
	"strings"
	"sync"
)

var errCredentialFileEmpty = errors.New("credential file empty")

type credentialFileSignature struct {
	modTimeUnixNano int64
	size            int64
}

func signatureForFileInfo(info os.FileInfo) credentialFileSignature {
	return credentialFileSignature{
		modTimeUnixNano: info.ModTime().UnixNano(),
		size:            info.Size(),
	}
}

type cachedCredentialFile struct {
	sig   credentialFileSignature
	value string
}

type credentialFileCache struct {
	mu      sync.Mutex
	entries map[string]cachedCredentialFile
}

func newCredentialFileCache() *credentialFileCache {
	return &credentialFileCache{entries: make(map[string]cachedCredentialFile)}
}

func (cache *credentialFileCache) readTrimmed(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	sig := signatureForFileInfo(info)

	cache.mu.Lock()
	entry, ok := cache.entries[path]
	cache.mu.Unlock()

	if ok {
		if entry.sig == sig && entry.value != "" {
			return entry.value, nil
		}
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(string(buf))
	if value == "" {
		return "", errCredentialFileEmpty
	}

	cache.mu.Lock()
	cache.entries[path] = cachedCredentialFile{sig: sig, value: value}
	cache.mu.Unlock()

	return value, nil
}

var credentialFiles = newCredentialFileCache()
