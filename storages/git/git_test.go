package git

import (
	"os"
	"path/filepath"
	"testing"
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
