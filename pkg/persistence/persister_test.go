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
	Context("leveldb persister", func() {
		p := persistence.NewLevelDBPersister(os.TempDir())
		errChan := p.Errors()

		It("persists a goyaad job and then recovers it", func(done Done) {
			defer close(done)

			j := goyaad.NewJobAutoID(time.Now(), testBody)
			data, err := j.GobEncode()
			Expect(err).To(BeNil())
			inputEntry := &persistence.Entry{Data: bytes.NewBuffer(data), Namespace: "simpletest"}
			err = p.Persist(inputEntry)
			Expect(err).To(BeNil())

			Expect(errChan).ShouldNot(Receive())

			p.Finalize()

			// recovers a job
			entries, err := p.Recover("simpletest")
			Expect(err).To(BeNil())
			var item interface{}
			Eventually(entries).Should(Receive(&item))
			var entry *persistence.Entry
			Expect(entry).To(BeAssignableToTypeOf(entry))
			entry = item.(*persistence.Entry)

			Expect(entry.Namespace).To(Equal(inputEntry.Namespace))
			Expect(entry.Data.Bytes()).To(BeEquivalentTo(data))

			jj := new(goyaad.Job)
			err = jj.GobDecode(entry.Data.Bytes())
			Expect(err).To(BeNil())
			Expect(jj.Body()).To(Equal(j.Body()))
			Expect(jj.ID()).To(Equal(j.ID()))
			Expect(jj.TriggerAt().UnixNano()).To(Equal(j.TriggerAt().UnixNano()))
		}, 5)
	})
})
