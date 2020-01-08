package persistence

import (
	"fmt"
	"io"
)

// Storage provides the underlying data store used by the persister
type Storage interface {
	// Reset deletes any data stored in the storage
	Reset() error
	// Writer creates a new io.WriteCloser for the storage
	Writer() (io.WriteCloser, error)
	// Reader creates a new io.ReadCloser for the storage
	Reader() (io.ReadCloser, error)

	fmt.Stringer

	// verifyAccess to actual storage - better to check access at startup and fail rather than just before saving data
	verifyAccess() error
}
