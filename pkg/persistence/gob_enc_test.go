package persistence_test

import (
	"encoding/gob"
	"io"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test gob persistence", func() {
	It("tests gob persistence", func() {
		jc := make(chan gob.GobEncoder, 3)
		j := goyaad.NewJobAutoID(time.Now(), []byte(`helloworld`))
		jc <- j
		j = goyaad.NewJobAutoID(time.Now(), []byte(`helloworld2`))
		jc <- j
		close(jc)

		gp := persistence.GobPersister{}
		err := gp.Store(jc)
		Expect(err).ToNot(HaveOccurred())

		// read
		dec, err := gp.Decoder()
		Expect(err).ToNot(HaveOccurred())
		for {
			var j = &goyaad.Job{}
			err := j.Decode(dec)
			if err != nil {
				if err == io.EOF {
					break
				}
				Expect(err).ToNot(HaveOccurred())
			}
			log.Error().Str("jobbody", string(j.Body())).Send()
		}
	})
})
