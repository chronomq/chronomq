package persistence

import (
	"bytes"
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
	stream             chan *Entry // Internal stream so that all writes are ordered
	dataDir            string
	namespaceWriterMap map[Namespace]*journal.Writer

	finalize chan struct{}

	entryStats map[Namespace]int
}

// NewJournalPersister initializes a Journal backed persister
func NewJournalPersister(dataDir string) Persister {
	lp := &JournalPersister{
		stream:             make(chan *Entry, 10),
		dataDir:            dataDir,
		namespaceWriterMap: make(map[Namespace]*journal.Writer),
		finalize:           make(chan struct{}, 1),
		entryStats:         make(map[Namespace]int),
	}

	logrus.Infof("Created Journal persister with datadir: %s", dataDir)
	return lp
}

// ResetDataDir tells persister to *delete* everything in the datadir
func (lp *JournalPersister) ResetDataDir() error {
	// Currently, only reset namespaces
	p := lp.getPathForNamespace("")
	logrus.Warnf("JournalPersister:ResetDataDir resetting base path: %s ", p)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		dir, err := ioutil.ReadDir(p)
		if err != nil {
			return err
		}
		for _, d := range dir {
			loc := path.Join(p, d.Name())
			logrus.WithField("Path", loc).Warnf("JournalPersister:ResetDataDir Resetting file %s", d.Name())
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
func (lp *JournalPersister) Finalize() {
	logrus.Info("JournalPersister:Finalize finalizing persister")

	// close db
	for ns, w := range lp.namespaceWriterMap {
		logrus.WithField("Namespace", ns).Info("JournalPersister:Finalize closing writer db")
		err := w.Flush()
		if err != nil {
			logrus.Error("JournalPersister:Finalize error flushing journal", err)
		}
		err = w.Close()
		if err != nil {
			logrus.Error("JournalPersister:Finalize error finalizing writer", err)
		}
		logrus.WithField("Namespace", ns).Info("JournalPersister:Finalize closed writer db")
	}
	logrus.Info("JournalPersister:Finalize closed all db namespaces")
	logrus.WithField("Stats", lp.entryStats).Infof("JournalPersister:writer persister stopped %+v", lp.entryStats)
}

// Persist stores an entry to disk
func (lp *JournalPersister) Persist(e *Entry) error {
	logrus.Debug("JournalPersister:Persist persisting an entry")
	return lp.write(e)
}

// PersistStream listens to the input channel and persists entries to disk
func (lp *JournalPersister) PersistStream(ec chan *Entry) chan error {
	errC := make(chan error)
	go func() {
		defer close(errC)
		for e := range ec {
			logrus.Debug("JournalPersister:PersistStream persisting an entry")
			if err := lp.write(e); err != nil {
				errC <- err
			}
		}
	}()
	return errC
}

// Recover reads back persisted data and emits entries
func (lp *JournalPersister) Recover(namespace Namespace) (chan *Entry, error) {
	logger := logrus.WithFields(logrus.Fields{"Namespace": namespace})

	entriesChan := make(chan *Entry)

	filePath := lp.getPathForNamespace(namespace)
	logger.WithField("File", filePath).Infof("JournalPersister:Recover starting recovery")
	f, err := os.Open(filePath)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence file for Namespace: "+namespace)
		logger.Infof("JournalPersister:Recover %s", err)
		return nil, err
	}
	r := journal.NewReader(f, nil, false, true)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence file for Namespace: "+namespace)
		logger.Infof("JournalPersister:Recover %s", err)
		return nil, err
	}

	logger.Infof("JournalPersister:Recover streaming items for recovery")
	go func() {
		defer close(entriesChan)

		for {
			j, err := r.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = errors.Wrap(err, "Failed to fetch next journal reader for Namespace: "+namespace)
				logrus.Errorf("JournalPersister:Recover %s", err)
				break
			}
			buf, err := ioutil.ReadAll(j)
			if err != nil {
				logrus.Tracef("JournalPersister:Recover error reading from journal %s", err)
			}
			e := &Entry{
				Data:      bytes.NewBuffer(buf),
				Namespace: namespace,
			}
			entriesChan <- e
		}
		logger.Infof("JournalPersister:Recover finished recovery stream")
	}()

	return entriesChan, nil
}

func (lp *JournalPersister) write(e *Entry) error {

	jw, ok := lp.namespaceWriterMap[e.Namespace]
	if !ok {
		var err error
		filePath := lp.getPathForNamespace(e.Namespace)
		logrus.WithField("File", filePath).Infof("JournalPersister:writer starting persistence")
		f, err := os.Create(filePath)
		if err != nil {
			err = errors.Wrap(err, "JournalPersister:writer Failed to open peristence file for Namespace: "+e.Namespace)
			logrus.Error(err)
			return err
		}
		jw = journal.NewWriter(f)
		lp.namespaceWriterMap[e.Namespace] = jw
		lp.entryStats[e.Namespace] = 0
	}

	w, err := jw.Next()
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to get next journal writer")
		logrus.Error(err)
		return err
	}
	_, err = w.Write(e.Data.Bytes())
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to persist entry")
		logrus.Error(err)
		return err
	}
	lp.entryStats[e.Namespace] = lp.entryStats[e.Namespace] + 1
	return nil
}

func (lp *JournalPersister) getPathForNamespace(n Namespace) string {
	p := path.Join(lp.dataDir, "namespaces")
	os.MkdirAll(p, os.ModeDir|0774)
	return path.Join(p, n)
}
