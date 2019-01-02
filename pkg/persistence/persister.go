package persistence

// Persister saves the data given to it to a durable data store like a disk, S3 buckets, durable streams etc
type Persister interface {
	ResetDataDir() error

	Persist(buf []byte) error
	PersistStream(bufC chan []byte) chan error
	Finalize()

	Recover() (chan []byte, error)
}
