package persistence

import (
	"encoding/gob"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/journal"
)

// JournalPersister saves data in an embedded Journal store
type JournalPersister struct {
	stream  chan gob.GobEncoder // Internal stream so that all writes are ordered
	dataDir string
	writer  *journal.Writer

	finalize chan struct{}
}

// NewJournalPersister initializes a Journal backed persister
func NewJournalPersister(dataDir string) Persister {
	lp := &JournalPersister{
		stream:   make(chan gob.GobEncoder, 10),
		dataDir:  dataDir,
		writer:   nil, // lazy init writer
		finalize: make(chan struct{}, 1),
	}

	logrus.Infof("Created Journal persister with datadir: %s", dataDir)
	return lp
}

// ResetDataDir tells persister to *delete* everything in the datadir
func (lp *JournalPersister) ResetDataDir() error {
	// Currently, only reset namespaces
	p := lp.getPath()
	logrus.Warnf("JournalPersister:ResetDataDir resetting base path: %s", p)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		return os.Remove(p)
	}

	return nil
}

// Finalize tells persister that it can finalize and close writes
// It is an error to send new items to persist once Finalize has been called
func (lp *JournalPersister) Finalize() {
	logrus.Info("JournalPersister:Finalize finalizing persister")

	// close db
	if lp.writer != nil {
		logrus.Info("JournalPersister:Finalize closing writer db")
		err := lp.writer.Flush()
		if err != nil {
			logrus.Error("JournalPersister:Finalize error flushing journal", err)
		}
		err = lp.writer.Close()
		if err != nil {
			logrus.Error("JournalPersister:Finalize error finalizing writer", err)
		}
		logrus.Info("JournalPersister:Finalize closed writer db")
	}
	logrus.Info("JournalPersister:Finalize done")
}

// Persist stores an entry to disk
func (lp *JournalPersister) Persist(enc gob.GobEncoder) error {
	logrus.Debug("JournalPersister:Persist persisting an entry")
	return lp.write(enc)
}

// PersistStream listens to the input channel and persists entries to disk
func (lp *JournalPersister) PersistStream(encC chan gob.GobEncoder) chan error {
	errC := make(chan error)
	go func() {
		defer close(errC)
		for e := range encC {
			logrus.Debug("JournalPersister:PersistStream persisting an entry")

			if err := lp.write(e); err != nil {
				errC <- err
				continue
			}
		}
	}()
	return errC
}

// Recover reads back persisted data and emits entries
func (lp *JournalPersister) Recover() (chan []byte, error) {
	bufC := make(chan []byte)

	filePath := lp.getPath()
	logrus.WithField("File", filePath).Infof("JournalPersister:Recover starting recovery")
	f, err := os.Open(filePath)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence file")
		logrus.Errorf("JournalPersister:Recover %s", err)
		return nil, err
	}
	r := journal.NewReader(f, nil, false, true)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence journal reader")
		logrus.Errorf("JournalPersister:Recover %s", err)
		return nil, err
	}

	logrus.Info("JournalPersister:Recover streaming items for recovery")
	go func() {
		defer close(bufC)

		for {
			j, err := r.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = errors.Wrap(err, "Failed to fetch next journal reader")
				logrus.Errorf("JournalPersister:Recover %s", err)
				break
			}
			buf, err := ioutil.ReadAll(j)
			if err != nil {
				logrus.Tracef("JournalPersister:Recover error reading from journal %s", err)
				continue
			}
			bufC <- buf
		}
		logrus.Infof("JournalPersister:Recover finished recovery stream")
	}()

	return bufC, nil
}

func (lp *JournalPersister) write(enc gob.GobEncoder) error {

	// lazy init writer
	if lp.writer == nil {
		var err error
		filePath := lp.getPath()
		logrus.WithField("File", filePath).Infof("JournalPersister:writer starting persistence")
		f, err := os.Create(filePath)
		if err != nil {
			err = errors.Wrap(err, "JournalPersister:writer Failed to open peristence file")
			logrus.Error(err)
			return err
		}
		lp.writer = journal.NewWriter(f)
	}

	w, err := lp.writer.Next()
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to get next journal writer")
		logrus.Error(err)
		return err
	}

	buf, err := enc.GobEncode()
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to persist entry")
		logrus.Error(err)
		return err
	}
	return nil
}

// getPathForNamespace returns the canonical path where data for the given namespace will be persisted
func (lp *JournalPersister) getPath() string {
	p := path.Join(lp.dataDir, "journal")
	os.MkdirAll(p, os.ModeDir|0774)
	return path.Join(p, "jobs.snapshot")
}
