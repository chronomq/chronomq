package persistence

import (
	"bytes"
	"io/ioutil"
	"os"
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

// ResetDataDir tells persister to *delete* everything in the datadir
func (lp *LevelDBPersister) ResetDataDir() error {
	// Currently, only reset namespaces
	p := lp.getPathForNamespace("")
	logrus.Warnf("LevelDBPersister:ResetDataDir resetting base path: %s ", p)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		dir, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		for _, d := range dir {
			loc := path.Join(p, d.Name())
			logrus.WithField("Path", loc).Warnf("LevelDBPersister:ResetDataDir Resetting file %s", d.Name())
			err = os.RemoveAll(loc)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
	logger := logrus.WithFields(logrus.Fields{"Namespace": namespace})

	ec := make(chan *Entry)
	filePath := lp.getPathForNamespace(namespace)
	logger.WithField("File", filePath).Infof("LevelDBPersister:Recover starting recovery")
	db, err := leveldb.OpenFile(filePath, nil)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence file for Namespace: "+namespace)
		logger.Infof("LevelDBPersister:Recover %s", err)
		return nil, err
	}

	logger.Infof("LevelDBPersister:Recover streaming items for recovery")
	go func() {
		defer db.Close()
		defer close(ec)

		iter := db.NewIterator(nil, nil)
		defer iter.Release()

		for iter.Next() {
			value := iter.Value()
			e := &Entry{
				Data:      bytes.NewBuffer(value),
				Namespace: namespace,
			}
			ec <- e
		}
		logger.Infof("LevelDBPersister:Recover finished recovery stream")
	}()

	return ec, nil
}

func (lp *LevelDBPersister) writer() {
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
				filePath := lp.getPathForNamespace(e.Namespace)
				logrus.WithField("File", filePath).Infof("LevelDBPersister:writer starting persistence")
				db, err = leveldb.OpenFile(filePath, nil)
				if err != nil {
					err = errors.Wrap(err, "LevelDBPersister:writer Failed to open peristence file for Namespace: "+e.Namespace)
					logrus.Error(err)
					lp.errChan <- err
					continue
				}
				lp.namespaceDBMap[e.Namespace] = db
			}

			err := db.Put([]byte(e.Key), e.Data.Bytes(), nil)
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

func (lp *LevelDBPersister) getPathForNamespace(n Namespace) string {
	return path.Join(lp.dataDir, "namespaces", n)
}
