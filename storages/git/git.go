package git

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	sshGit "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/crypto/ssh"
)

// authHTTP discovers environment for HTTP credentials
func authBasicHTTP() (*http.BasicAuth, error) {
	username, okUsername := os.LookupEnv("GIT_USERNAME")
	if !okUsername {
		return nil, errors.New("Git protocol was http but username was not set")
	}

	token, okToken := os.LookupEnv("GIT_TOKEN")
	if !okToken {
		password, okPassword := os.LookupEnv("GIT_PASSWORD")
		if !okPassword {
			return nil, errors.New("Git protocol was http but neither password nor token was set")
		}
		token = password
	}

	return &http.BasicAuth{
		Username: username,
		Password: token,
	}, nil
}

// authSSH discovers environment for SSH credentials
func authSSH() (*sshGit.PublicKeys, error) {
	// If SSH_AUTH_SOCK env variable was defined - nothing else needs to be configured
	if _, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok {
		return nil, nil
	}

	pemFile, okPem := os.LookupEnv("SSH_PRIVATE_KEY")
	if !okPem {
		// Ok then, try to discover SSH keys in the user home
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		pemFile = home + "/.ssh/id_rsa"
	}

	pem, err := ioutil.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, err
	}

	return &sshGit.PublicKeys{User: "git", Signer: signer}, nil
}

// auth discovers Git authentification in the environment
func auth(params *RequestMetadataParams) (transport.AuthMethod, error) {
	// If protocol was HTTP, try to discover Basic auth methods from the environment
	if strings.HasPrefix(params.Repository, "http") {
		// Otherwise we assume protocol was SSH.
		auth, err := authBasicHTTP()
		if err != nil {
			return nil, err
		}

		return auth, nil
	}

	// Otherwise we assume protocol was SSH.
	auth, err := authSSH()
	if err != nil {
		return nil, err
	}

	return auth, nil
}

// ref convert short branch name string to a full ReferenceName
func ref(branch string, remote bool) plumbing.ReferenceName {
	var ref string
	if remote {
		ref = "refs/remotes/origin/"
	} else {
		ref = "refs/heads/"
	}
	return plumbing.ReferenceName(ref + branch)
}

// newStorageSession makes a fresh clone to in-memory FS and saves everything to the StorageSession
func newStorageSession(params *RequestMetadataParams) (*storageSession, error) {
	storageSession := &storageSession{
		storer: memory.NewStorage(),
		fs:     memfs.New(),
		mutex:  sync.Mutex{},
	}

	if err := storageSession.clone(params); err != nil {
		return nil, err
	}

	return storageSession, nil
}

// clone remote repository
func (storageSession *storageSession) clone(params *RequestMetadataParams) error {
	auth, err := auth(params)
	if err != nil {
		return err
	}

	storageSession.auth = auth

	cloneOptions := &git.CloneOptions{
		URL:           params.Repository,
		Auth:          auth,
		ReferenceName: ref(params.Ref, false),
	}

	repository, err := git.Clone(storageSession.storer, storageSession.fs, cloneOptions)
	if err != nil {
		return err
	}

	storageSession.repository = repository

	return nil
}

// getRemote returns "origin" remote.
// Since we never specified a name for our remote, it should always be origin.
func (storageSession *storageSession) getRemote() (*git.Remote, error) {
	remote, err := storageSession.repository.Remote("origin")
	if err != nil {
		return nil, err
	}

	return remote, nil
}

// CheckoutMode configures checkout behaviour
type CheckoutMode uint8

const (
	// CheckoutModeDefault is default checkout mode - no special behaviour
	CheckoutModeDefault CheckoutMode = 1 << iota
	// CheckoutModeCreate will indicate that the new local branch needs to be created at checkout
	CheckoutModeCreate
	// CheckoutModeRemote will indicate that the remote branch needs to be checked out
	CheckoutModeRemote
)

// checkout this repository working copy to specified branch.
// If create flag was true, it will make an attempt to create a new branch and it will return an error if it already existed.
func (storageSession *storageSession) checkout(branch string, mode CheckoutMode) error {
	if mode&CheckoutModeCreate != 0 && mode&CheckoutModeRemote != 0 {
		return errors.New("CheckoutModeCreate and CheckoutModeRemote cannot be used simultaniously")
	}

	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}

	checkoutOptions := &git.CheckoutOptions{
		Branch: ref(branch, mode&CheckoutModeRemote != 0),
		Force:  true,
		Create: mode&CheckoutModeCreate != 0,
	}

	if err := tree.Checkout(checkoutOptions); err != nil {
		return err
	}

	return nil
}

// Attempt to pull from remote to the current branch.
// This branch must already exist locally and upstream must be set for it to know where to pull from.
// It will ignore git.NoErrAlreadyUpToDate.
func (storageSession *storageSession) pull(branch string) error {
	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}

	pullOptions := git.PullOptions{
		ReferenceName: ref(branch, false),
		Force:         true,
		Auth:          storageSession.auth,
	}

	if err := tree.Pull(&pullOptions); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

var (
	locksRefSpecs = []config.RefSpec{
		"refs/heads/locks/*:refs/remotes/origin/locks/*",
	}
)

// Attempt to fetch from remote for specified ref specs.
// It will ignore git.NoErrAlreadyUpToDate.
func (storageSession *storageSession) fetch(refs []config.RefSpec) error {
	fetchOptions := git.FetchOptions{
		RefSpecs: refs,
		Auth:     storageSession.auth,
	}

	remote, err := storageSession.getRemote()
	if err != nil {
		return err
	}

	if err := remote.Fetch(&fetchOptions); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

// Will delete the branch locally.
// Additionally delete branch remotely if deleteRemote was set true.
// Operation is idempotent, i.e. no error will be returned if the branch did not existed.
func (storageSession *storageSession) deleteBranch(branch string, deleteRemote bool) error {
	ref := ref(branch, false)

	if err := storageSession.repository.Storer.RemoveReference(ref); err != nil {
		return err
	}

	if !deleteRemote {
		return nil
	}

	remote, err := storageSession.getRemote()
	if err != nil {
		return err
	}

	pushOptions := &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(":" + ref),
		},
		Auth: storageSession.auth,
	}

	if err := remote.Push(pushOptions); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	return nil
}

// add path to the local working tree
func (storageSession *storageSession) add(path string) error {
	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}

	if _, err := tree.Add(path); err != nil {
		return err
	}

	return nil
}

// delete path from the local working tree
func (storageSession *storageSession) delete(path string) error {
	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}

	if _, err := tree.Remove(path); err != nil {
		return err
	}

	return nil
}

// commit currently staged changes to the local working tree
func (storageSession *storageSession) commit(msg string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	host, err := os.Hostname()
	if err != nil {
		return err
	}

	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}

	commitOptions := git.CommitOptions{
		Author: &object.Signature{
			Name:  user.Name,
			Email: user.Username + "@" + host,
			When:  time.Now(),
		},
	}

	if _, err := tree.Commit(msg, &commitOptions); err != nil {
		return err
	}

	return nil
}

// push current working tree state to the remote repository
// It assumes the upstream has been set for the current branch - it will not do anything to define the ref.
func (storageSession *storageSession) push() error {
	remote, err := storageSession.getRemote()
	if err != nil {
		return err
	}

	pushOptions := git.PushOptions{
		Auth: storageSession.auth,
	}

	if err := remote.Push(&pushOptions); err != nil {
		return err
	}

	return nil
}

// fileExists returns true if file existed in the working tree
func (storageSession *storageSession) fileExists(path string) (bool, error) {
	info, err := storageSession.fs.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return !info.IsDir(), nil
}

// readFile reads a file in the local working tree.
func (storageSession *storageSession) readFile(path string) ([]byte, error) {
	var buf []byte

	file, err := storageSession.fs.Open(path)
	if err != nil {
		return buf, err
	}
	defer file.Close()

	return ioutil.ReadAll(file)
}

// writeFile write this buf to the file in the local working tree.
// Either new file will be created or existing one gets overwritten.
func (storageSession *storageSession) writeFile(path string, buf []byte) error {
	file, err := storageSession.fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, bytes.NewReader(buf)); err != nil {
		return err
	}

	return nil
}
