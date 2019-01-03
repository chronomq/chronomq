package persistence

import "encoding/gob"

// Persister saves the data given to it to a durable data store like a disk, S3 buckets, durable streams etc
type Persister interface {
	ResetDataDir() error

	Persist(enc gob.GobEncoder) error
	PersistStream(encC chan gob.GobEncoder) chan error
	Finalize()

	Recover() (chan []byte, error)
}
