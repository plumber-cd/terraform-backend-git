//go:build !windows
// +build !windows

package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/plumber-cd/terraform-backend-git/backend"
	"github.com/plumber-cd/terraform-backend-git/types"
)

type reloadTestClient struct {
	tokenFile string
	updates   chan string
}

func (c *reloadTestClient) Reload() error {
	buf, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return err
	}
	c.updates <- strings.TrimSpace(string(buf))
	return nil
}

func (c *reloadTestClient) ParseMetadataParams(*http.Request, *types.RequestMetadata) error {
	return nil
}

func (c *reloadTestClient) Connect(types.RequestMetadataParams) error {
	return nil
}

func (c *reloadTestClient) Disconnect(types.RequestMetadataParams) {}

func (c *reloadTestClient) LockState(types.RequestMetadataParams, []byte) error {
	return nil
}

func (c *reloadTestClient) ReadStateLock(types.RequestMetadataParams) ([]byte, error) {
	return nil, nil
}

func (c *reloadTestClient) UnLockState(types.RequestMetadataParams) error {
	return nil
}

func (c *reloadTestClient) ForceUnLockWorkaroundMessage(types.RequestMetadataParams) string {
	return ""
}

func (c *reloadTestClient) GetState(types.RequestMetadataParams) ([]byte, error) {
	return nil, nil
}

func (c *reloadTestClient) UpdateState(types.RequestMetadataParams, []byte) error {
	return nil
}

func (c *reloadTestClient) DeleteState(types.RequestMetadataParams) error {
	return nil
}

func TestSIGHUPReload_InvokesReloadableStorageClients(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("tok1\n"), 0600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	client := &reloadTestClient{
		tokenFile: tokenFile,
		updates:   make(chan string, 2),
	}

	saved := make(map[string]types.StorageClient, len(backend.KnownStorageTypes))
	for k, v := range backend.KnownStorageTypes {
		saved[k] = v
	}
	t.Cleanup(func() {
		for k := range backend.KnownStorageTypes {
			delete(backend.KnownStorageTypes, k)
		}
		for k, v := range saved {
			backend.KnownStorageTypes[k] = v
		}
	})

	for k := range backend.KnownStorageTypes {
		delete(backend.KnownStorageTypes, k)
	}
	backend.KnownStorageTypes["test"] = client

	startReloadHandler()

	sendHUP := func() {
		if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
			t.Fatalf("send SIGHUP: %v", err)
		}
	}

	awaitToken := func(want string) {
		t.Helper()
		select {
		case got := <-client.updates:
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for reload")
		}
	}

	sendHUP()
	awaitToken("tok1")

	if err := os.WriteFile(tokenFile, []byte("tok2\n"), 0600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	sendHUP()
	awaitToken("tok2")
}
