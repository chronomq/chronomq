package persistence

import (
	"encoding/gob"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

	log.Info().Str("datadir", dataDir).Msg("Created Journal persister with datadir")
	return lp
}

// ResetDataDir tells persister to *delete* everything in the datadir
func (lp *JournalPersister) ResetDataDir() error {
	// Currently, only reset namespaces
	p := lp.getPath()
	log.Warn().Str("basePath", p).Msg("JournalPersister:ResetDataDir resetting base path")
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		return os.Remove(p)
	}

	return nil
}

// Finalize tells persister that it can finalize and close writes
// It is an error to send new items to persist once Finalize has been called
func (lp *JournalPersister) Finalize() {
	log.Info().Msg("JournalPersister:Finalize finalizing persister")

	// close db
	if lp.writer != nil {
		log.Info().Msg("JournalPersister:Finalize closing writer db")
		err := lp.writer.Flush()
		if err != nil {
			log.Error().Err(err).Msg("JournalPersister:Finalize error flushing journal")
		}
		err = lp.writer.Close()
		if err != nil {
			log.Error().Err(err).Msg("JournalPersister:Finalize error finalizing writer")
		}
		log.Info().Msg("JournalPersister:Finalize closed writer db")
	}
	log.Info().Msg("JournalPersister:Finalize done")
}

// Persist stores an entry to disk
func (lp *JournalPersister) Persist(enc gob.GobEncoder) error {
	log.Debug().Msg("JournalPersister:Persist persisting an entry")
	return lp.write(enc)
}

// PersistStream listens to the input channel and persists entries to disk
func (lp *JournalPersister) PersistStream(encC chan gob.GobEncoder) chan error {
	errC := make(chan error)
	go func() {
		defer close(errC)
		for e := range encC {
			log.Debug().Msg("JournalPersister:PersistStream persisting an entry")

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
	log.Info().Str("file", filePath).Msg("JournalPersister:Recover starting recovery")
	f, err := os.Open(filePath)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence file")
		log.Error().Err(err).Msg("JournalPersister:Recover")
		return nil, err
	}
	r := journal.NewReader(f, nil, false, true)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence journal reader")
		log.Error().Err(err).Msg("JournalPersister:Recover")
		return nil, err
	}

	log.Info().Msg("JournalPersister:Recover streaming items for recovery")
	go func() {
		defer close(bufC)

		for {
			j, err := r.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = errors.Wrap(err, "Failed to fetch next journal reader")
				log.Error().Err(err).Msg("JournalPersister:Recover")
				break
			}
			buf, err := ioutil.ReadAll(j)
			if err != nil {
				log.Debug().Err(err).Msg("JournalPersister:Recover error reading from journal")
				continue
			}
			bufC <- buf
		}
		log.Info().Msg("JournalPersister:Recover finished recovery stream")
	}()

	return bufC, nil
}

func (lp *JournalPersister) write(enc gob.GobEncoder) error {

	// lazy init writer
	if lp.writer == nil {
		var err error
		filePath := lp.getPath()
		log.Info().Str("file", filePath).Msg("JournalPersister:writer starting persistence")
		f, err := os.Create(filePath)
		if err != nil {
			err = errors.Wrap(err, "JournalPersister:writer Failed to open peristence file")
			log.Error().Err(err).Send()
			return err
		}
		lp.writer = journal.NewWriter(f)
	}

	w, err := lp.writer.Next()
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to get next journal writer")
		log.Error().Err(err).Send()
		return err
	}

	buf, err := enc.GobEncode()
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	if err != nil {
		err = errors.Wrap(err, "JournalPersister:writer failed to persist entry")
		log.Error().Err(err).Send()
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
