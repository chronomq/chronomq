// +build wasm

package persistence

import "encoding/gob"

type NoopPersister struct{}

func (np *NoopPersister) ResetDataDir() error { return nil }

func (np *NoopPersister) Persist(gob.GobEncoder) error { return nil }

func (np *NoopPersister) PersistStream(chan gob.GobEncoder) chan error { return make(chan error) }

func (np *NoopPersister) Finalize() {}

func (np *NoopPersister) Recover() (chan []byte, error) { return make(chan []byte), nil }
