package persistence

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"path"
	"time"

	s3util "github.com/rlmcpherson/s3gof3r"
	"github.com/rs/zerolog/log"
)

// S3StoreConfig - config for s3 backed storage
type S3StoreConfig struct {
	Bucket string
	Prefix string
}

// s3 provides access to S3 as a storage layer for persistence
type s3 struct {
	key    string
	bucket *s3util.Bucket
}

// NewS3Store creates a new s3 backed store
func NewS3Store(cfg S3StoreConfig) (Storage, error) {
	s3util.SetLogger(log.Logger, "", stdlog.LstdFlags, true)
	keys, err := s3util.EnvKeys()
	if err != nil {
		log.Error().Err(err).Msg("S3Store::create Unable to find AWS S3 keys from the Env")
		return nil, err
	}
	s3u := s3util.New("", keys)
	b := s3u.Bucket(cfg.Bucket)

	key := path.Join(cfg.Prefix, "journal", "jobs.snapshot")
	s := &s3{key: key, bucket: b}
	return s, s.verifyAccess()
}

// Reset deletes any data stored in the storage
func (s *s3) Reset() error {
	return s.bucket.Delete(s.key)
}

// Writer creates a new io.Writer for the storage
func (s *s3) Writer() (io.WriteCloser, error) {
	log.Info().Msg("S3Store::Writer creating new writer")
	return s.writer(s.key)
}

// writer creates a writer for the given key location
func (s *s3) writer(key string) (io.WriteCloser, error) {
	w, err := s.bucket.PutWriter(key, nil, nil)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Reader creates a new io.Reader for the storage
func (s *s3) Reader() (io.ReadCloser, error) {
	log.Info().Msg("S3Store::Reader creating new reader")
	return s.reader(s.key)
}

// reader creates a new io.Reader for the storage
func (s *s3) reader(key string) (io.ReadCloser, error) {
	// Ignore the headers
	r, _, err := s.bucket.GetReader(key, nil)
	if err != nil {
		log.Error().Err(err).Msg("S3Store:reader failed to create a reader")
	}
	return r, err
}

// verifyAccess makes sure the store is wired correctly reachable
// Detect access failures as early as possible rather than at shutdown much later
func (s *s3) verifyAccess() error {
	log.Info().Msg("verifying access for s3")
	testKey := s.key + ".test"
	testData := []byte(`access_check__` + time.Now().String())
	// Write some test data
	w, err := s.writer(testKey)
	if err != nil {
		log.Error().Err(err).Msg("Store:s3:accessCheck Failed create writer")
		return err
	}
	if _, err = w.Write(testData); err != nil {
		log.Error().Err(err).Msg("Store:s3:accessCheck Failed to write sentinel")
		return err
	}
	if err = w.Close(); err != nil {
		log.Error().Err(err).Msg("Store:s3:accessCheck Failed to close writer")
		return err
	}

	// Read it back to verify we have full access
	r, _, err := s.bucket.GetReader(testKey, nil)
	if err != nil {
		log.Error().Err(err).Msg("Store:s3:accessCheck Failed to create reader")
		return err
	}
	readData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Error().Err(err).Msg("Store:s3:accessCheck Failed to write read sentinel")
		return err
	}
	if !bytes.Equal(readData, testData) {
		err = errors.New("Unable to verify s3 store data integrity")
		log.Error().
			Str("read", string(readData)).
			Str("wrote", string(testData)).
			Err(err).Msg("Store:s3:accessCheck Failed integrity check")
		return err
	}
	return nil
}

func (s *s3) String() string {
	return fmt.Sprintf("s3://%s/%s", s.bucket.Name, s.key)
}
