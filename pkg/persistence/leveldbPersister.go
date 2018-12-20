package persistence

import (
	"fmt"
	"path"

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

	writerClosed chan struct{}
	finalize     chan struct{}
}

// NewLevelDBPersister initializes a LevelDB backed persister
func NewLevelDBPersister(dataDir string) Persister {
	lp := &LevelDBPersister{
		stream:         make(chan *Entry, 10),
		dataDir:        dataDir,
		namespaceDBMap: make(map[Namespace]*leveldb.DB),
		errChan:        make(chan error, 10),
		finalize:       make(chan struct{}, 1),
		writerClosed:   make(chan struct{}, 1),
	}
	go lp.writer()

	return lp
}

// Finalize tells persister that it can finalize and close writes
// It is an error to send new items to persist once Finalize has been called
func (lp *LevelDBPersister) Finalize() {
	logrus.Info("LevelDBPersister:Finalize finalizing persister")
	lp.finalize <- struct{}{}
	<-lp.writerClosed
}

// Persist stores an entry to disk
func (lp *LevelDBPersister) Persist(e *Entry) error {
	logrus.Debug("LevelDBPersister:Persist persisting an entry")
	lp.stream <- e
	return nil
}

// PersistStream listens to the input channel and persists entries to disk
func (lp *LevelDBPersister) PersistStream(ec chan *Entry) error {
	for e := range ec {
		logrus.Debug("LevelDBPersister:PersistStream persisting an entry")
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
	var counterKey uint64

	logrus.Info("LevelDBPersister:writer Starting leveldb persister writer process")

	for lp.stream != nil {
		select {
		case e, ok := <-lp.stream:
			if !ok {
				lp.stream = nil
				break
			}

			db, ok := lp.namespaceDBMap[e.Namespace]
			if !ok {
				var err error
				db, err = leveldb.OpenFile(path.Join(lp.dataDir, e.Namespace), nil)
				if err != nil {
					err = errors.Wrap(err, "LevelDBPersister:writer Failed to open peristence file for Namespace: "+e.Namespace)
					logrus.Error(err)
					lp.errChan <- err
					continue
				}
				lp.namespaceDBMap[e.Namespace] = db
			}

			err := db.Put([]byte(fmt.Sprintf("%d", counterKey)), e.Data.Bytes(), nil)
			if err != nil {
				err = errors.Wrap(err, "LevelDBPersister:writer failed to persist entry")
				logrus.Error(err)
				lp.errChan <- err
				continue
			}

		case _, ok := <-lp.finalize:
			if !ok {
				break
			}
			logrus.Info("LevelDBPersister:writer finalizing persister")
			for ns, db := range lp.namespaceDBMap {
				logrus.WithField("Namespace", ns).Info("LevelDBPersister:writer closing writer db")
				err := db.Close()
				if err != nil {
					logrus.Info("LevelDBPersister:writer error finalizing persister")
				}
				logrus.WithField("Namespace", ns).Info("LevelDBPersister:writer closed writer db")
			}
			logrus.Info("LevelDBPersister:writer closed all db namespaces")
			// Cant receive finalize again
			close(lp.finalize)
			// Cant receive on persist stream now
			close(lp.stream)

			// Tell caller of finalize that we are done cleaning up
			lp.writerClosed <- struct{}{}
			close(lp.writerClosed)
		}
	}
	logrus.Info("LevelDBPersister:writer ended writer")
}
