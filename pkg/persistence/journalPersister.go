// +build !wasm

package persistence

import (
	"encoding/gob"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/syndtr/goleveldb/leveldb/journal"
)

// JournalPersister saves data in an embedded Journal store
type JournalPersister struct {
	stream  chan gob.GobEncoder // Internal stream so that all writes are ordered
	storage Storage
	writer  *journal.Writer

	// leveldb journal Writer sadly doesn't propage close to the underlying writer
	// so we are keeping a reference here to close it on Finalize. We could probably
	// move to using Gob streams directly on underlying storage
	storeWriter io.Closer
}

// NewJournalPersister initializes a Journal backed persister
func NewJournalPersister(s Storage) Persister {
	lp := &JournalPersister{
		stream:  make(chan gob.GobEncoder, 10),
		storage: s,
		writer:  nil,
	}

	log.Info().Str("store", s.String()).Msg("Created Journal persister with store")
	return lp
}

// ResetDataDir tells persister to *delete* everything in the datadir
func (lp *JournalPersister) ResetDataDir() error {
	return lp.storage.Reset()
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
			log.Error().Err(err).Msg("JournalPersister:Finalize error closing journal writer")
		}
	}

	// close storage writer
	if lp.storeWriter != nil {
		err := lp.storeWriter.Close()
		if err != nil {
			log.Error().Err(err).Msg("JournalPersister:Finalize error closing store writer")
		}
	}
	log.Info().Msg("JournalPersister:Finalize done")
}

// Persist stores an entry to given storage
func (lp *JournalPersister) Persist(enc gob.GobEncoder) error {
	log.Debug().Msg("JournalPersister:Persist persisting an entry")
	err := lp.write(enc)
	if err != nil {
		log.Error().Err(err).Send()
	}
	return err
}

// PersistStream listens to the input channel and persists entries to storage
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
	log.Info().Msg("JournalPersister:Recover starting recovery")

	// Get a reader from the store
	sr, err := lp.storage.Reader()
	if err != nil {
		err = errors.Wrap(err, "Failed to open store")
		log.Error().Err(err).Msg("JournalPersister:Recover")
		return nil, err
	}
	// Get a new journal reader wrapping the store reader
	r := journal.NewReader(sr, nil, false, true)
	if err != nil {
		err = errors.Wrap(err, "Failed to open peristence journal reader")
		log.Error().Err(err).Msg("JournalPersister:Recover")
		return nil, err
	}

	log.Info().Msg("JournalPersister:Recover streaming items for recovery")
	bufC := make(chan []byte)
	go func() {
		defer close(bufC)
		// Close storage reader
		defer sr.Close()

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
	// lazy init journal writer
	if lp.writer == nil {
		var err error
		log.Info().Msg("JournalPersister:writer starting persistence")
		sw, err := lp.storage.Writer()
		if err != nil {
			err = errors.Wrap(err, "JournalPersister:writer Failed to open store")
			log.Error().Err(err).Send()
			return err
		}
		lp.storeWriter = sw
		lp.writer = journal.NewWriter(sw)
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
