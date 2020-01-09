package persistence

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

// StoreType - named store implementations
type StoreType = string

const (
	// FileStore selects a file system backed storage
	FileStore StoreType = "FS"
	// S3Store selects an s3 backed storage
	S3Store StoreType = "S3"
)

// StoreConfig for storage
type StoreConfig struct {
	Store StoreType

	S3Cfg S3StoreConfig
	FSCfg FSStoreConfig
}

// Storage creates a new Storage based on the config
func (cfg *StoreConfig) Storage() (Storage, error) {
	log.Info().Msg("Creating store: " + cfg.Store)
	switch cfg.Store {
	case FileStore:
		return NewFSStore(cfg.FSCfg)
	case S3Store:
		return NewS3Store(cfg.S3Cfg)
	default:
		return nil, errors.New("Unknown store type option")
	}
}
