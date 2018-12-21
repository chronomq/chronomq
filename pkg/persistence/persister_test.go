package persistence_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var testBody = []byte("Hello world")

var _ = Describe("Test persistence", func() {

	Context("leveldb persister", func() {
		persistenceTestDir := path.Join(os.TempDir(), "goyaadtest")
		var p persistence.Persister
		var errChan chan error

		BeforeEach(func() {
			p = persistence.NewLevelDBPersister(persistenceTestDir)
			errChan = p.Errors()
			Expect(p.ResetDataDir()).To(BeNil())
		})

		It("properly resets data dir", func() {
			j := goyaad.NewJobAutoID(time.Now(), testBody)
			data, err := j.GobEncode()
			Expect(err).To(BeNil())
			inputEntry := &persistence.Entry{Data: bytes.NewBuffer(data), Namespace: "simpletest"}
			err = p.Persist(inputEntry)
			Expect(err).To(BeNil())

			Expect(errChan).ShouldNot(Receive())

			p.Finalize()

			// reset
			Expect(p.ResetDataDir()).To(BeNil())

			dir, err := ioutil.ReadDir(path.Join(persistenceTestDir, "namespaces"))
			Expect(err).To(BeNil())
			Expect(len(dir)).To(Equal(0))
		})

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
			entriesChan, err := p.Recover("simpletest")
			Expect(err).To(BeNil())

			// Wait for entries chan to close
			entries := []*persistence.Entry{}
			for e := range entriesChan {
				entries = append(entries, e)
			}

			Expect(len(entries)).To(Equal(1))
			entry := entries[0]

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
