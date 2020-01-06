package persistence

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	s3util "github.com/rlmcpherson/s3gof3r"
	"github.com/rs/zerolog/log"
)

// slim s3 bucket interface to help testing
type bucket interface {
	Delete(key string) error
	PutWriter(path string, h http.Header, c *s3util.Config) (w io.WriteCloser, err error)
	GetReader(path string, c *s3util.Config) (r io.ReadCloser, h http.Header, err error)
}

// s3 provides access to S3 as a storage layer for persistence
type s3 struct {
	bucketName string
	b          bucket
	key        string
}

// NewS3 creates a new s3 backed store
// supply a custom http client if need. If customClient is nil,
func NewS3(bucketName, prefix string) (Storage, error) {
	keys, err := s3util.EnvKeys()
	if err != nil {
		return nil, err
	}
	s3u := s3util.New("", keys)
	b := s3u.Bucket(bucketName)
	s3util.SetLogger(os.Stdout, "", 1, true)
	key := path.Join(prefix, "journal", "jobs.snapshot")
	s := &s3{bucketName, b, key}
	return s, s.setup()
}

// Reset deletes any data stored in the storage
func (s *s3) Reset() error {
	return s.b.Delete(s.key)
}

// Path returns the canonical storage location for s3 including the final key
func (s *s3) path() string {
	return fmt.Sprintf("%s/%s", s.bucketName, s.key)
}

// Writer creates a new io.Writer for the storage
func (s *s3) Writer() (io.WriteCloser, error) {
	return s.writer(s.key)
}

// writer creates a writer for the given key location
func (s *s3) writer(key string) (io.WriteCloser, error) {
	log.Info().Msg("s3store ::: creating writer for key " + key)
	w, err := s.b.PutWriter(key, nil, nil)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Reader creates a new io.Reader for the storage
func (s *s3) Reader() (io.ReadCloser, error) {
	return s.reader(s.key)
}

// reader creates a new io.Reader for the storage
func (s *s3) reader(key string) (io.ReadCloser, error) {
	// Ignore the headers
	r, _, err := s.b.GetReader(key, nil)
	if err != nil {
		log.Error().Err(err).Msg("Store:s3:reader failed to create a reader")
	}
	return r, err
}

// AccessCheck makes sure the store is wired correctly reachable
// Detect access failures as early as possible rather than at shutdown much later
func (s *s3) setup() error {
	testKey := s.key + ".test"
	testData := []byte(`access_check__` + time.Now().String())
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
	time.Sleep(time.Second)
	r, _, err := s.b.GetReader(testKey, nil)
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
	return "s3://" + s.path()
}
