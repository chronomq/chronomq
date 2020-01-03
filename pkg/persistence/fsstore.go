package persistence

import (
	"io"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// fs provides access to the local file-system as a storage layer for persistence
type fs struct {
	dataDir string
	w       io.WriteCloser
}

// NewFS creates a new file based storage
func NewFS(loc string) (Storage, error) {
	dataDir := path.Join(loc, "journal")
	log.Info().Msg("Store:fs Creating file store")
	f := &fs{dataDir: dataDir}
	log.Info().Msgf("store: %s", f)
	err := f.setup()
	return f, err
}

// Reset deletes any data stored in the storage
func (f *fs) Reset() error {
	log.Warn().Str("basePath", f.path()).Msg("Store:fs:reset resetting storage location")
	if _, err := os.Stat(f.path()); !os.IsNotExist(err) {
		return os.Remove(f.path())
	}
	return nil
}

// path returns the canonical storage location
func (f *fs) path() string {
	return path.Join(f.dataDir, "jobs.snapshot")
}

// Writer creates a new io.Writer for the storage
func (f *fs) Writer() (io.Writer, error) {
	if f.w == nil {
		w, err := os.OpenFile(f.path(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0600))
		if err != nil {
			err = errors.Wrap(err, "Store:fs:writer Failed to open file")
			log.Error().Err(err).Send()
			return nil, err
		}
		log.Info().Str("filename", w.Name()).Msg("Store:fs:writer Created writer")
		f.w = w
	}
	return f.w, nil
}

// Reader creates a new io.Reader for the storage
func (f *fs) Reader() (io.Reader, error) {
	log.Info().Str("file", f.path()).Msg("Store:fs:reader setting up file store")
	r, err := os.Open(f.path())
	if err != nil {
		err = errors.Wrap(err, "Store:fs:reader failed to open file")
		log.Error().Err(err).Send()
		return nil, err
	}
	return r, err
}

// Close the storage provider
func (f *fs) Close() error {
	if f.w != nil {
		return f.w.Close()
	}
	return nil
}

// Setup makes sure the store is wired correctly reachable
// detects access failures as early as possible rather than at shutdown much later
func (f *fs) setup() error {
	// Check that we have r/w access to datadir
	dirMode := os.FileMode(os.ModeDir | 0700)
	err := os.MkdirAll(f.dataDir, dirMode)
	if err != nil {
		err = errors.Wrap(err, "Store:fs:setup Failed to setup data dir")
		log.Error().Err(err).Send()
		return err
	}
	// explicitly assert mode (mkdirAll early returns without applying perms if dir exists)
	dirInfo, err := os.Stat(f.dataDir)
	if err != nil {
		err = errors.Wrap(err, "Store:fs:setup Failed to stat data dir")
		log.Error().Err(err).Send()
		return err
	}
	if dirInfo.Mode()&dirMode == 0 {
		err = errors.New("Store:fs:setup data dir has bad permissions")
		log.Error().Err(err).Str("loc", f.dataDir).Send()
		return err
	}
	// at this point, data dir has right permissions and exists
	return nil
}

func (f *fs) String() string {
	return f.path()
}
