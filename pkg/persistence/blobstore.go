package persistence

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/azureblob" // azure blob store
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob" // google bucket
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob" // aws s3 bucket
)

// StoreConfig - config for data store
type StoreConfig struct {
	Bucket *url.URL
}

var verifyAccessKey = "accesscheck"
var dataKey = "jobs.snapshot"

type blobStore struct {
	bucket *blob.Bucket
	cfg    StoreConfig
}

// NewBlobStore creates a new blob Storage
func NewBlobStore(cfg StoreConfig) (Storage, error) {
	ctx := context.Background()
	b, err := blob.OpenBucket(ctx, cfg.Bucket.String())
	if err != nil {
		return nil, err
	}
	s := &blobStore{
		bucket: blob.PrefixedBucket(b, "journal/"),
		cfg:    cfg,
	}

	err = s.verifyAccess()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (b *blobStore) Writer() (io.WriteCloser, error) {
	return b.bucket.NewWriter(context.Background(), dataKey, nil)
}

func (b *blobStore) Reader() (io.ReadCloser, error) {
	return b.bucket.NewReader(context.Background(), dataKey, nil)
}

func (b *blobStore) Reset() error {
	ok, err := b.bucket.Exists(context.Background(), dataKey)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return b.bucket.Delete(context.Background(), dataKey)
}

func (b *blobStore) String() string {
	return b.cfg.Bucket.String()
}

func (b *blobStore) verifyAccess() error {
	wd := []byte(`access_check__` + time.Now().String())
	err := b.bucket.WriteAll(context.Background(), verifyAccessKey, wd, nil)
	if err != nil {
		return err
	}
	rd, err := b.bucket.ReadAll(context.Background(), verifyAccessKey)
	if err != nil {
		return err
	}
	if !bytes.Equal(wd, rd) {
		err = errors.New("Store:blob:verifyAccess data integrity check failed")
		log.Error().Err(err).Send()
		return err
	}
	return b.bucket.Delete(context.Background(), verifyAccessKey)
}
