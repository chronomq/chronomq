package persistence_test

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/yaad/pkg/yaad"
	"github.com/urjitbhatia/yaad/pkg/persistence"
)

var testBody = []byte("Hello world")

var _ = Describe("Test persistence", func() {

	Context("persister functions", func() {
		persistenceTestDir := path.Join(os.TempDir(), "yaadtest")
		var p persistence.Persister

		BeforeEach(func() {
			p = persistence.NewJournalPersister(persistenceTestDir)
			Expect(p.ResetDataDir()).To(BeNil())
		})

		It("properly resets data dir", func() {
			j := yaad.NewJobAutoID(time.Now(), testBody)
			err := p.Persist(j)
			Expect(err).To(BeNil())

			p.Finalize()

			// reset
			Expect(p.ResetDataDir()).To(BeNil())

			dir, err := ioutil.ReadDir(path.Join(persistenceTestDir, "journal"))
			Expect(err).To(BeNil())
			Expect(len(dir)).To(Equal(0))
		})

		It("persists a yaad job and then recovers it", func(done Done) {
			defer close(done)

			j := yaad.NewJobAutoID(time.Now(), testBody)
			err := p.Persist(j)
			Expect(err).To(BeNil())

			p.Finalize()

			// recovers a job
			jobsChan, err := p.Recover()
			Expect(err).To(BeNil())

			// Wait for entries chan to close
			jobs := []yaad.Job{}
			for buf := range jobsChan {
				j := yaad.Job{}
				err = j.GobDecode(buf)
				Expect(err).To(BeNil())
				jobs = append(jobs, j)
			}

			Expect(len(jobs)).To(Equal(1))
			job := jobs[0]
			Expect(job.Body()).To(Equal(j.Body()))
			Expect(job.ID()).To(Equal(j.ID()))
			Expect(job.TriggerAt().UnixNano()).To(Equal(j.TriggerAt().UnixNano()))
		}, 5)
	})
})
