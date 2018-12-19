package persistence

// LevelDBPersister saves data to a LevelDB file on disk
type LevelDBPersister struct{}

// NewLevelDBPersister initializes a LevelDB persister
func NewLevelDBPersister() Persister {
	return &LevelDBPersister{}
}

// Persist stores an entry to LevelDB
func (dp *LevelDBPersister) Persist(e *Entry) error {
	return nil
}

// PersistStream listens to the input channel and persists entries to LevelDB
func (dp *LevelDBPersister) PersistStream(ec chan<- *Entry) error {
	return nil
}
