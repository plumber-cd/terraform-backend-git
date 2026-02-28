package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

func TestAuthBasicHTTP_EnvPassword(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")
	t.Setenv("GIT_PASSWORD", "pass")

	auth, err := authBasicHTTP()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Username != "user" {
		t.Fatalf("expected username 'user', got %q", auth.Username)
	}
	if auth.Password != "pass" {
		t.Fatalf("expected password 'pass', got %q", auth.Password)
	}
}

func TestAuthBasicHTTP_PasswordFile(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")

	file := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(file, []byte("fromfile\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("GIT_PASSWORD_FILE", file)

	auth, err := authBasicHTTP()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Password != "fromfile" {
		t.Fatalf("expected password 'fromfile', got %q", auth.Password)
	}
}

func TestAuthBasicHTTP_GithubTokenFile(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")

	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("tok\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("GITHUB_TOKEN_FILE", file)

	auth, err := authBasicHTTP()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Password != "tok" {
		t.Fatalf("expected password 'tok', got %q", auth.Password)
	}
}

func TestAuthBasicHTTP_EnvOverridesFile(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")

	file := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(file, []byte("fromfile\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("GIT_PASSWORD_FILE", file)
	t.Setenv("GIT_PASSWORD", "fromenv")

	auth, err := authBasicHTTP()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if auth.Password != "fromenv" {
		t.Fatalf("expected password 'fromenv', got %q", auth.Password)
	}
}

func TestAuthBasicHTTP_MissingUsername(t *testing.T) {
	_, err := authBasicHTTP()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestStorageSessionRemoteAuth_RefetchesTokenFile(t *testing.T) {
	t.Setenv("GIT_USERNAME", "user")

	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("tok1\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	t.Setenv("GITHUB_TOKEN_FILE", file)

	s := &storageSession{remoteURL: "https://example.invalid/repo.git"}

	auth1, err := s.remoteAuth()
	if err != nil {
		t.Fatalf("remoteAuth: %v", err)
	}
	ba1, ok := auth1.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("expected BasicAuth, got %T", auth1)
	}
	if ba1.Password != "tok1" {
		t.Fatalf("expected password %q, got %q", "tok1", ba1.Password)
	}

	if err := os.WriteFile(file, []byte("tok2\n"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	// Some filesystems have a coarse mtime resolution; ensure the stat signature changes.
	now := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(file, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	auth2, err := s.remoteAuth()
	if err != nil {
		t.Fatalf("remoteAuth: %v", err)
	}
	ba2, ok := auth2.(*githttp.BasicAuth)
	if !ok {
		t.Fatalf("expected BasicAuth, got %T", auth2)
	}
	if ba2.Password != "tok2" {
		t.Fatalf("expected password %q, got %q", "tok2", ba2.Password)
	}
}
