package persistence

import "bytes"

// Namespace groups data to be persisted
type Namespace = string

// Entry is a simple wrapper that needs to be persisted
type Entry struct {
	// Data is any arbitrary data that gets persisted
	Data *bytes.Buffer
	// Namespace groups things that will be persisted together
	// For example, Deletes can all be persisted together etc
	Namespace Namespace
}

// Persister saves the data given to it to a durable data store like a disk, S3 buckets, durable streams etc
type Persister interface {
	Persist(e *Entry) error
	PersistStream(ec chan *Entry) error
	Finalize()

	Recover(namespace Namespace) (chan *Entry, error)

	Errors() chan error
}
