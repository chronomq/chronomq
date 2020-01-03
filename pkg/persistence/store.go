package persistence

import (
	"fmt"
	"io"
)

// Storage provides the underlying data store used by the persister
type Storage interface {
	// Reset deletes any data stored in the storage
	Reset() error
	// Writer creates a new io.Writer for the storage
	Writer() (io.Writer, error)
	// Reader creates a new io.Reader for the storage
	Reader() (io.Reader, error)
	// Close the store
	Close() error

	fmt.Stringer
}
