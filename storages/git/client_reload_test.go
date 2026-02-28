package git

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

func TestStorageClientReload_UpdatesCachedSessionAuth(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")

	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("tok1\n"), 0600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	t.Setenv("GITHUB_TOKEN_FILE", tokenFile)

	repo := "https://example.invalid/repo.git"

	client := &StorageClient{
		sessions: map[string]*storageSession{
			repo: {
				auth:  &githttp.BasicAuth{Username: "user", Password: "old"},
				mutex: sync.Mutex{},
			},
		},
		sessionsMutex: sync.Mutex{},
	}

	if err := client.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	auth, ok := client.sessions[repo].auth.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("expected BasicAuth, got %T", client.sessions[repo].auth)
	}
	if auth.Password != "tok1" {
		t.Fatalf("expected password %q, got %q", "tok1", auth.Password)
	}

	if err := os.WriteFile(tokenFile, []byte("tok2\n"), 0600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	if err := client.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	auth, ok = client.sessions[repo].auth.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("expected BasicAuth, got %T", client.sessions[repo].auth)
	}
	if auth.Password != "tok2" {
		t.Fatalf("expected password %q, got %q", "tok2", auth.Password)
	}
}
