package persistence_test

import (
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/urjitbhatia/goyaad/internal/job"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var testBody = []byte("Hello world")

var _ = Describe("Test persistence", func() {

	Context("persister functions", func() {
		bucketDirPath := path.Join(os.TempDir(), "goyaadtest")
		testDirPath := path.Join(bucketDirPath, "journal")
		storeURL := &url.URL{Scheme: "file", Path: bucketDirPath}

		var p persistence.Persister

		BeforeSuite(func() {
			err := os.MkdirAll(testDirPath, os.ModeDir|os.FileMode(0777))
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeEach(func() {
			store, err := persistence.StoreConfig{Bucket: storeURL}.Storage()
			Expect(err).ToNot(HaveOccurred())
			p = persistence.NewJournalPersister(store)
			Expect(p.ResetDataDir()).To(BeNil())
		})

		It("properly resets data dir", func() {
			j := job.NewJobAutoID(time.Now(), testBody)
			err := p.Persist(j)
			Expect(err).To(BeNil())

			p.Finalize()

			// reset
			err = p.ResetDataDir()
			Expect(err).To(BeNil())

			dir, err := ioutil.ReadDir(testDirPath)
			Expect(err).To(BeNil(), testDirPath)
			Expect(len(dir)).To(Equal(0), testDirPath)
		})

		It("persists a goyaad job and then recovers it", func(done Done) {
			defer close(done)

			j := job.NewJobAutoID(time.Now(), testBody)
			err := p.Persist(j)
			Expect(err).To(BeNil())

			p.Finalize()

			// recovers a job
			jobsChan, err := p.Recover()
			Expect(err).To(BeNil())

			// Wait for entries chan to close
			jobs := []job.Job{}
			for buf := range jobsChan {
				j := job.Job{}
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
