package persistence

import (
	"encoding/gob"
	"os"
)

type GobPersister struct{}

func (gp *GobPersister) Store(e chan gob.GobEncoder) error {
	f, err := os.OpenFile("/tmp/mygob.bog", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(f)
	for v := range e {
		err := enc.Encode(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gp *GobPersister) Decoder() (*gob.Decoder, error) {
	f, err := os.OpenFile("/tmp/mygob.bog", os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}
	return gob.NewDecoder(f), nil

	// jc := make(chan interface{}, 2)
	// go func() {
	// 	var j interface{}
	// 	err := dec.Decode(&j)
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			log.Info().Msg("End of decode stream")
	// 		}
	// 		log.Fatal().Err(err).Send()
	// 	}
	// 	jc <- j
	// }()
	// return jc, nil
}
