package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/plumber-cd/terraform-backend-git/types"
	"github.com/spf13/viper"
)

// KnownStorageTypes map storage types to storage clients before starting the server so backend knows what's supported
var KnownStorageTypes = make(map[string]types.StorageClient)

// ParseMetadata look into the request and read metadata
func ParseMetadata(request *http.Request) (*types.RequestMetadata, error) {

	metadata := &types.RequestMetadata{
		Type: viper.GetString("backend"),
	}

	if request.URL.Query().Get("type") != "" {
		metadata.Type = request.URL.Query().Get("type")
	}

	metadata.ID = request.URL.Query().Get("ID")

	if metadata.Type == "" {
		return nil, errors.New("Storage type was not specified")
	}

	return metadata, nil
}

// GetStorageClient initialize and return a StorageClient.
func GetStorageClient(metadata *types.RequestMetadata) (types.StorageClient, error) {
	if storageClient, ok := KnownStorageTypes[metadata.Type]; ok {
		return storageClient, nil
	}

	return nil, fmt.Errorf("Unknown storage type %s", metadata.Type)
}

// lockedByMe trying to read the lock from the storage and check if it's locked by the requestor.
// ReadStateLock implementations must return ErrLockMissing if it didn't exist.
func lockedByMe(metadata *types.RequestMetadata, storageClient types.StorageClient) error {
	lock, err := storageClient.ReadStateLock(metadata.Params)
	if err != nil {
		return err
	}

	var lockInfo types.LockInfo
	if err := json.Unmarshal(lock, &lockInfo); err != nil {
		return err
	}

	if metadata.ID == lockInfo.ID {
		return nil
	}

	return &types.ErrLocked{
		Lock:     lock,
		LockInfo: &lockInfo,
	}
}

// LockState will lock the state as requested.
// Locking must be atomic operation so leave all checks for the client.
// Client implementations must return ErrLockingConflict if it was already locked by someone else.
func LockState(metadata *types.RequestMetadata, storageClient types.StorageClient, body []byte) error {
	if err := storageClient.LockState(metadata.Params, body); err != nil {
		// If it was a conflict, using lockedByMe here will return an ErrLocked since lock ID was missing in the request
		if err == types.ErrLockingConflict {
			if err := lockedByMe(metadata, storageClient); err != nil {
				return err
			}
		}
		return err
	}

	return nil
}

// UnLockState will unlock the state as requested.
func UnLockState(metadata *types.RequestMetadata, storageClient types.StorageClient, body []byte) error {
	// Assuming the proper fix for the broken force-unlock in HTTP TF backend is to set HTTP request parameter ID,
	// and it's presense will indicate that force-unlock has been used.
	force := metadata.ID != ""

	// If it wasn't force-unlock, there will be a request body with the lock metadata that originally locked this state
	if !force {
		var lock types.LockInfo
		if err := json.Unmarshal(body, &lock); err != nil {
			if err == io.EOF {
				log.Println(`WARNING: force-unlock is currently broken.
	Reason: https://github.com/hashicorp/terraform/blob/master/backend/remote-state/http/client.go is broken.
	Unlock function in HTTP TF backend does not using lockID.
	Our backend would never know the ID to unlock when force-unlock was used.
	` + storageClient.ForceUnLockWorkaroundMessage(metadata.Params))
			}
			return err
		}

		// We only interested in the Lock ID - make the backend think it was supplied as HTTP request paramater
		metadata.ID = lock.ID
	}

	if err := lockedByMe(metadata, storageClient); err != nil {
		return err
	}

	if err := storageClient.UnLockState(metadata.Params); err != nil {
		return err
	}

	return nil
}

// GetState attempt to read the state from storage.
// Clinet implementations must return NoErrStateDidNotExisted if the state did not existed.
func GetState(metadata *types.RequestMetadata, storageClient types.StorageClient) ([]byte, error) {
	state, err := storageClient.GetState(metadata.Params)
	if err != nil {
		return nil, err
	}

	stateDecrypted, err := decryptIfEnabled(state)
	if err != nil {
		return nil, err
	}

	return stateDecrypted, nil
}

// UpdateState create or update existing state.
// This is a write operation, so it's checking if the state was previously locked by a requestor.
func UpdateState(metadata *types.RequestMetadata, storageClient types.StorageClient, body []byte) error {
	if err := lockedByMe(metadata, storageClient); err != nil {
		return err
	}

	stateMaybeEncrypted, err := encryptIfEnabled(body)
	if err != nil {
		return err
	}

	if err := storageClient.UpdateState(metadata.Params, stateMaybeEncrypted); err != nil {
		return err
	}

	return nil
}

// DeleteState deleting state from the storage.
// This is a write operation, so it's checking if the state was previously locked by a requestor.
func DeleteState(metadata *types.RequestMetadata, storageClient types.StorageClient) error {
	if err := lockedByMe(metadata, storageClient); err != nil {
		return err
	}

	if err := storageClient.DeleteState(metadata.Params); err != nil {
		return err
	}

	return nil
}
