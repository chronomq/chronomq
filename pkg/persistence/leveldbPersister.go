package persistence

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	leveldb "github.com/syndtr/goleveldb/leveldb"
)

// LevelDBPersister saves data in an embedded leveldb store
type LevelDBPersister struct {
	stream         chan *Entry // Internal stream so that all writes are ordered
	dataDir        string
	namespaceDBMap map[Namespace]*leveldb.DB
	errChan        chan error

	finalize chan struct{}
}

// NewLevelDBPersister initializes a LevelDB backed persister
func NewLevelDBPersister(dataDir string) Persister {
	lp := &LevelDBPersister{
		stream:         make(chan *Entry, 10),
		dataDir:        dataDir,
		namespaceDBMap: make(map[Namespace]*leveldb.DB),
		errChan:        make(chan error, 10),
		finalize:       make(chan struct{}, 1),
	}
	go lp.writer()

	return lp
}

// Finalize tells persister that it can finalize and close writes
// It is an error to send new items to persist once Finalize has been called
func (lp *LevelDBPersister) Finalize() {
	logrus.Info("LevelDBPersister: finalizing")
	lp.finalize <- struct{}{}
}

// Persist stores an entry to disk
func (lp *LevelDBPersister) Persist(e *Entry) error {
	logrus.Info("LevelDBPersister/Persist: persisting an entry")
	lp.stream <- e
	return nil
}

// PersistStream listens to the input channel and persists entries to disk
func (lp *LevelDBPersister) PersistStream(ec chan *Entry) error {
	for e := range ec {
		logrus.Info("LevelDBPersister/Stream saving entry")
		lp.stream <- e
	}
	return nil
}

// Errors returns a channel that clients of this persister should listen on for errors
func (lp *LevelDBPersister) Errors() chan error {
	return lp.errChan
}

// Recover reads back persisted data and emits entries
func (lp *LevelDBPersister) Recover(namespace Namespace) (chan *Entry, error) {
	return make(chan *Entry), errors.New("not implemented")
}

func (lp *LevelDBPersister) writer() {
	logrus.Error("not implemented")
}
