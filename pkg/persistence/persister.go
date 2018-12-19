package persistence

// Entry is a simple wrapper that needs to be persisted
type Entry struct {
	// Data is any arbitrary interface that gets persisted
	Data interface{}
	// Namespace groups things that will be persisted together
	// For example, Deletes can all be persisted together etc
	Namespace string
}

// Persister saves the data given to it to a durable data store like a disk, S3 buckets, durable streams etc
type Persister interface {
	Persist(e *Entry) error
	PersistStream(ec chan<- *Entry) error
}
