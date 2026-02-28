package git

import (
	"fmt"
	"sync"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage"
)

// RequestMetadataParams is Git storage specific parameters
type RequestMetadataParams struct {
	Repository, Ref, State string
	Amend                  bool
}

// String is a human readable representation for this params set
func (params *RequestMetadataParams) String() string {
	return fmt.Sprintf("%s?ref=%s&amend=%t//%s", params.Repository, params.Ref, params.Amend, params.State)
}

// StorageClient implementation for Git storage type
type StorageClient struct {
	// sessions key is repository URL, value is everything we need to interact with it
	sessions map[string]*storageSession

	// sessionsMutex used for locking sessions map for adding new repositories
	sessionsMutex sync.Mutex
}

// storageSession represents a particular Git repository
type storageSession struct {
	// remoteURL is the git repository URL used for remote operations
	remoteURL string

	// storer used for local working tree config
	storer storage.Storer

	// fs can be used to access local working tree
	fs billy.Filesystem

	// repository represents a git repository
	repository *git.Repository

	// mutex since we can't be doing parallel complex operations on a single working tree, involving checkout branches and etc,
	// we need to use the lock and make sure only one tread is "connected" (interacts with the repository usingl local working tree).
	mutex sync.Mutex
}
