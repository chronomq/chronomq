package persistence_test

import (
	"bytes"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var testBody = []byte("Hello world")

var _ = Describe("Test persistence", func() {
	Context("With a leveldb persister", func() {
		p := persistence.NewLevelDBPersister(os.TempDir())

		It("for goyaad job entry", func() {
			j := goyaad.NewJobAutoID(time.Now(), testBody)
			data, err := j.GobEncode()
			Expect(err).To(BeNil())

			err = p.Persist(&persistence.Entry{Data: bytes.NewBuffer(data), Namespace: "test"})
			Expect(err).To(BeNil())
		})
	})
})
