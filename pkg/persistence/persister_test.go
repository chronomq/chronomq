package persistence_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var testBody = []byte("Hello world")

var _ = Describe("Test persistence", func() {
	Context("With a leveldb persister", func() {
		p := persistence.NewLevelDBPersister()

		It("for goyaad job entry", func() {
			j := goyaad.NewJobAutoID(time.Now(), testBody)
			err := p.Persist(&persistence.Entry{Data: j, Namespace: "test"})
			Expect(err).To(BeNil())
		})
	})
})
