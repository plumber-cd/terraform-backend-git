package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	sshagent "github.com/xanzy/ssh-agent"

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

	"github.com/plumber-cd/terraform-backend-git/backend"
)

func init() {
	backend.KnownStorageTypes["git"] = NewStorageClient()
}

// authHTTP discovers environment for HTTP credentials
func authBasicHTTP() (*http.BasicAuth, error) {
	username, okUsername := os.LookupEnv("GIT_USERNAME")
	if !okUsername {
		return nil, errors.New("Git protocol was http but username was not set")
	}

	password, okPassword := os.LookupEnv("GIT_PASSWORD")
	if !okPassword {
		ghToken, okGhToken := os.LookupEnv("GITHUB_TOKEN")
		if !okGhToken {
			return nil, errors.New("Git protocol was http but neither password nor token was set")
		}
		password = ghToken
	}

	return &http.BasicAuth{
		Username: username,
		Password: password,
	}, nil
}

// authSSHAgent discovers environment for SSH_AUTH_SOCK (or other platform dependent logic)
// and builds NewSSHAgentAuth.
//
// If returned null - no agent was set up
func authSSHAgent(params *RequestMetadataParams) (*sshGit.PublicKeysCallback, error) {
	if !sshagent.Available() {
		return nil, nil
	}

	e, err := transport.NewEndpoint(params.Repository)
	if err != nil {
		return nil, err
	}

	return sshGit.NewSSHAgentAuth(e.User)
}

// authSSH discovers environment for SSH credentials
func authSSH() (*sshGit.PublicKeys, error) {
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

	// Otherwise we assume protocol was SSH

	// Most likely strict known hosts checking not needed, but not making any assumptions
	strictHostKeyChecking := true
	hostKeyCallbackHelper := sshGit.HostKeyCallbackHelper{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if val, ok := os.LookupEnv("StrictHostKeyChecking"); ok && val == "no" {
		strictHostKeyChecking = false
	}

	// First, try ssh agent
	agent, err := authSSHAgent(params)
	if err != nil {
		return nil, err
	}
	if agent != nil {
		if !strictHostKeyChecking {
			agent.HostKeyCallbackHelper = hostKeyCallbackHelper
		}
		return agent, nil
	}

	// Otherwise, try to find some ssh keys
	key, err := authSSH()
	if err != nil {
		return nil, err
	}

	if !strictHostKeyChecking {
		key.HostKeyCallbackHelper = hostKeyCallbackHelper
	}

	return key, nil
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
		// We only need to know the latest version of branches to be able to commit on top
		// of them. And, we don't otherwise use the history of the repository. So, this
		// saves a lot data.
		// A further improvement that has not been implemented would be to use
		// sparse-checkouts to only retrieve only blobs (i.e., files) from the server that
		// we actually care about. The depth here prevents performance problems from
		// history growth. sparse-checkouts help with horizontal growth (e.g., additional
		// systems managed by the repository).
		Depth:		   1,
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
	return storageSession.commitWithOptions(msg, git.CommitOptions{})
}

// commit --amend currently staged changes to the local working tree
func (storageSession *storageSession) commitAmend(msg string) error {
	plumbing, _ := storageSession.repository.Head()
	return storageSession.commitWithOptions(fmt.Sprintf("%s (amendment of %s)", msg, plumbing.Hash().String()), git.CommitOptions{Amend: true})
}

func (storageSession *storageSession) commitWithOptions(msg string, opts git.CommitOptions) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	userName := user.Name
	if userName == "" {
		userName = user.Username
	}

	host, err := os.Hostname()
	if err != nil {
		return err
	}

	opts.Author = &object.Signature{
		Name:  userName,
		Email: user.Username + "@" + host,
		When:  time.Now(),
	}

	tree, err := storageSession.repository.Worktree()
	if err != nil {
		return err
	}
	if _, err := tree.Commit(msg, &opts); err != nil {
		return err
	}

	return nil
}

// push current working tree state to the remote repository
// It assumes the upstream has been set for the current branch - it will not do anything to define the ref.
func (storageSession *storageSession) push() error {
	return storageSession.pushWithOptions(git.PushOptions{})
}

func (storageSession *storageSession) pushForce() error {
	return storageSession.pushWithOptions(git.PushOptions{Force: true})
}

func (storageSession *storageSession) pushWithOptions(opts git.PushOptions) error {
	remote, err := storageSession.getRemote()
	if err != nil {
		return err
	}

	opts.Auth = storageSession.auth
	return remote.Push(&opts)
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
