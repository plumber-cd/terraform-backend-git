package git

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/plumber-cd/terraform-backend-git/types"
	"github.com/spf13/viper"
)

// NewStorageClient creates new StorageClient
func NewStorageClient() types.StorageClient {
	return &StorageClient{
		sessions:      make(map[string]*storageSession),
		sessionsMutex: sync.Mutex{},
	}
}

// ParseMetadataParams read request parameters specific to Git storage type
func (storageClient *StorageClient) ParseMetadataParams(request *http.Request, metadata *types.RequestMetadata) error {
	query := request.URL.Query()

	params := RequestMetadataParams{
		Repository: viper.GetString("git.repository"),
		Ref:        viper.GetString("git.ref"),
		State:      viper.GetString("git.state"),
	}

	if query.Get("repository") != "" {
		params.Repository = query.Get("repository")
	}

	if query.Get("ref") != "" {
		params.Ref = query.Get("ref")
	}

	if query.Get("state") != "" {
		params.State = filepath.Clean(query.Get("state"))
	}

	if params.Repository == "" {
		return errors.New("Missing parameter 'repository'")
	}

	if params.Ref == "" {
		params.Ref = "master"
	}

	if params.State == "" {
		return errors.New("Missing parameter 'state'")
	}

	metadata.Params = &params

	return nil
}

// Connect will clone this git repository to a virtual in-memory FS (or use previosly cloned cache),
// and put an exclusive lock (mutex, not TF lock) on it.
//
// This StorageClient implementation will use go-git and virtual in-memory FS as git local working tree.
// Each unique RequestMetadataParams.Repository will receive it's own in-memory FS instance.
// That FS will be shared among different requests to the same repository,
// and it will live in memory until backend restarts.
// This is to speed up various git actions and avoid fresh clone every time, which might be time consuming.
//
// Since simple TF-level actions like update state or lock/unlock in this implementation
// involves complex add-commit-push routines that aren't really an atomic operations,
// and the local working tree is shared by multiple requests to the same git repository,
// parallel requests can mess up the local working tree and leave it in broken state.
//
// Example - while one tree checked out the locking branch and preparing the lock metadata to write,
// another request came in for totally different state and it would checkout back to Ref and pull it from remote.
//
// Hence we just assume that Git type of storage does not support parallel connections to the same repository,
// and each connection will lock this repository from usage by other threads within the same backend instance.
//
// This is not currently implemented,
// but if the user values parallel connections feature over overall performance/memory requirements -
// we can use something else for the StorageClient.sessions map key.
// The locking/unlocking would still be required for thread-safety.
// That could be a configurable option.
func (storageClient *StorageClient) Connect(p types.RequestMetadataParams) error {
	params := p.(*RequestMetadataParams)

	storageClient.sessionsMutex.Lock()
	defer storageClient.sessionsMutex.Unlock()

	storageSession, ok := storageClient.sessions[params.Repository]
	if !ok {
		s, err := newStorageSession(params)
		if err != nil {
			return err
		}

		storageClient.sessions[params.Repository] = s
		storageSession = s
	}

	storageSession.mutex.Lock()

	return nil
}

// Disconnect from Git storage.
// There's nothing to "disconnect" really.
// We just need to unlock the local working copy for other threads.
func (storageClient *StorageClient) Disconnect(p types.RequestMetadataParams) {
	params := p.(*RequestMetadataParams)

	if storageSession, ok := storageClient.sessions[params.Repository]; ok {
		storageSession.mutex.Unlock()
	}
}

// LockState this implementation for Git storage will create and push a new branch to remote.
// The branch name will be the name of the state file prefixed by "locks/".
// Next to the state file in subject, there will be a ".lock" file added and commited, that will contain the lock metadata.
// If pushing that branch to remote fails (no fast-forward allowed),
// that would mean something else already aquired the lock before this.
// That approach would make a locking operation atomic.
//
// There's obviosly more than one way to implement the locking with Git.
// This implementation aims to avoid complex Git scenarios that would involve Git merges and dealing with Git conflicts.
// In other words, we are trying to keep the local working tree fast-forwardable at all times.
//
// And remember - git repository hosting the state is a "backend" storage and it's not meant to be used by people.
func (storageClient *StorageClient) LockState(p types.RequestMetadataParams, lock []byte) error {
	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.checkout(params.Ref, CheckoutModeDefault); err != nil {
		return err
	}

	if err := storageSession.pull(params.Ref); err != nil {
		return err
	}

	lockBranchName := getLockBranchName(params)

	// Delete any local leftowers from the past
	if err := storageSession.deleteBranch(lockBranchName, false); err != nil {
		return err
	}

	// Create local branch to start preparing a new lock metadata for push
	if err := storageSession.checkout(lockBranchName, CheckoutModeCreate); err != nil {
		return err
	}

	lockPath := getLockPath(params)

	if err := storageSession.writeFile(lockPath, lock); err != nil {
		return err
	}

	if err := storageSession.add(lockPath); err != nil {
		return err
	}

	if err := storageSession.commit("Lock " + params.State); err != nil {
		return err
	}

	if err := storageSession.push(); err != nil {
		// The lock already aquired by someone else
		if strings.HasPrefix(err.Error(), git.ErrNonFastForwardUpdate.Error()) {
			return types.ErrLockingConflict
		}

		return err
	}

	return nil
}

// ReadStateLock read the lock metadata from storage.
// This will fetch locks refs and try to checkout using remote lock branch.
// If it can't pull ("no reference found" error), means the lock didn't exist - ErrLockMissing returned.
// Otherwise it will read the lock metadata from remote HEAD and return it in buffer.
func (storageClient *StorageClient) ReadStateLock(p types.RequestMetadataParams) ([]byte, error) {
	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.fetch(locksRefSpecs); err != nil {
		return nil, err
	}

	lockBranchName := getLockBranchName(params)

	// Delete any local leftowers from the past
	if err := storageSession.deleteBranch(lockBranchName, false); err != nil {
		return nil, err
	}

	if err := storageSession.checkout(lockBranchName, CheckoutModeRemote); err != nil {
		return nil, err
	}

	if err := storageSession.pull(lockBranchName); err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, types.ErrLockMissing
		}
		return nil, err
	}

	lock, err := storageSession.readFile(getLockPath(params))
	if err != nil {
		return nil, err
	}

	return lock, nil
}

// UnLockState for Git storage type, unlocking is a simple branch deleting remotely
func (storageClient *StorageClient) UnLockState(p types.RequestMetadataParams) error {
	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.deleteBranch(getLockBranchName(params), true); err != nil {
		return err
	}

	return nil
}

// ForceUnLockWorkaroundMessage suggest the user to delete locking branch
func (storageClient *StorageClient) ForceUnLockWorkaroundMessage(p types.RequestMetadataParams) string {
	params := p.(*RequestMetadataParams)

	return fmt.Sprintf("As a workaround - please delete the branch %s manually in remote git repository %s.\n",
		getLockBranchName(params), params.Repository)
}

// GetState will checkout into Ref, pull the latest from remote, and try to read the state file from there.
// Will return ErrStateDidNotExisted if the state file did not existed.
func (storageClient *StorageClient) GetState(p types.RequestMetadataParams) ([]byte, error) {
	var state []byte

	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.checkout(params.Ref, CheckoutModeDefault); err != nil {
		return state, err
	}

	if err := storageSession.pull(params.Ref); err != nil {
		return state, err
	}

	state, err := storageSession.readFile(params.State)
	if err != nil {
		if err == os.ErrNotExist {
			return state, types.ErrStateDidNotExisted
		}
		return state, err
	}

	return state, nil
}

// UpdateState write the state to storage.
// It will checkout the Ref, pull the latest and try to add and commit the state in the request.
// The file in repository will either be created or overwritten.
func (storageClient *StorageClient) UpdateState(p types.RequestMetadataParams, state []byte) error {
	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.checkout(params.Ref, CheckoutModeDefault); err != nil {
		return err
	}

	if err := storageSession.pull(params.Ref); err != nil {
		return err
	}

	if err := storageSession.writeFile(params.State, state); err != nil {
		return err
	}

	if err := storageSession.add(params.State); err != nil {
		return err
	}

	if err := storageSession.commit("Update " + params.State); err != nil {
		return err
	}

	if err := storageSession.push(); err != nil {
		return err
	}

	return nil
}

// DeleteState delete the state from storage
// Checkout the Ref, pull the latest and attempt to delete the state file from there.
// Then commit and push.
func (storageClient *StorageClient) DeleteState(p types.RequestMetadataParams) error {
	params := p.(*RequestMetadataParams)

	storageSession := storageClient.sessions[params.Repository]

	if err := storageSession.checkout(params.Ref, CheckoutModeDefault); err != nil {
		return err
	}

	if err := storageSession.pull(params.Ref); err != nil {
		return err
	}

	if err := storageSession.delete(params.State); err != nil {
		return err
	}

	if err := storageSession.commit("Delete " + params.State); err != nil {
		return err
	}

	if err := storageSession.push(); err != nil {
		return err
	}

	return nil
}

// getLockPath calculates the path to a lock file
func getLockPath(params *RequestMetadataParams) string {
	return params.State + ".lock"
}

// getLockBranchName calculates the locking branch name
func getLockBranchName(params *RequestMetadataParams) string {
	return "locks/" + params.State
}
