package types

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrLocked indicates the state was already locked by someone else
type ErrLocked struct {
	Lock     []byte
	LockInfo *LockInfo
}

func (err *ErrLocked) Error() string {
	return fmt.Sprintf("the state was already locked by %s: %s", err.LockInfo.Who, err.LockInfo.ID)
}

var (
	// ErrStateDidNotExisted indicate that the state did not existed
	ErrStateDidNotExisted = errors.New("state did not existed")
	// ErrLockingConflict indicate the lock was already acquired by someone else
	ErrLockingConflict = errors.New("the lock was already acquired by someone else")
	// ErrLockMissing indicate that the lock didn't existed when it was expected/required to
	ErrLockMissing = errors.New("was not locked")
	// ErrUnauthorized indicates that the action was not authorized
	ErrUnauthorized = errors.New("Unauthorized")
)

// LockInfo represents a TF Lock Metadata.
// See https://github.com/hashicorp/terraform/blob/v1.1.3/internal/states/statemgr/locker.go#L115-L138.
// Thanks HashiCorp for using "internal" package :facepalm:.
type LockInfo struct {
	// Unique ID for the lock. NewLockInfo provides a random ID, but this may
	// be overridden by the lock implementation. The final value of ID will be
	// returned by the call to Lock.
	ID string

	// Terraform operation, provided by the caller.
	Operation string

	// Extra information to store with the lock, provided by the caller.
	Info string

	// user@hostname when available
	Who string

	// Terraform version
	Version string

	// Time that the lock was taken.
	Created time.Time

	// Path to the state file when applicable. Set by the Lock implementation.
	Path string
}

// RequestMetadataParams is a specific params set for a particular backend.
type RequestMetadataParams interface {
	// String is a human-readable representation for requested parameters.
	String() string
}

// RequestMetadata stores configuration passed from Terraform as HTTP request.
type RequestMetadata struct {
	ID, Type string
	Params   RequestMetadataParams
}

// StorageClient is a layer responsible for connection with the remote storage.
type StorageClient interface {
	// Parse HTTP request and read storage specific parameters - any error considered "bad request"
	ParseMetadataParams(*http.Request, *RequestMetadata) error

	// Connect to the remote storage and store connection in memory for this Params set
	Connect(RequestMetadataParams) error

	// Disconnect from remote storage if it was connected for this Params set.
	// Must not return any errors - there will be no one to handle them at disconnect.
	Disconnect(RequestMetadataParams)

	// Lock the state for current Params set.
	// Locking must be atomic operation.
	// Even though for the rest of requests checking the lock is a responsibility of backend (via ReadLock function),
	// the LockState should check if no one else has locked this state and lock it in the atomic way.
	// ErrLockingConflict will be returned if someone else got the lock.
	LockState(RequestMetadataParams, []byte) error

	// ReadStateLock current lock if it exists. Return ErrLockMissing if no lock was found.
	ReadStateLock(RequestMetadataParams) ([]byte, error)

	// Unlock currently locked state for current Params set.
	UnLockState(RequestMetadataParams) error

	// Since force-unlock is broken for HTTP TF backend, client implementtaion must suggest to the user how to workaround it.
	ForceUnLockWorkaroundMessage(RequestMetadataParams) string

	// Read state file from storage
	GetState(RequestMetadataParams) ([]byte, error)

	// Update state in the storage
	UpdateState(RequestMetadataParams, []byte) error

	// Delete state from the storage
	DeleteState(RequestMetadataParams) error
}
