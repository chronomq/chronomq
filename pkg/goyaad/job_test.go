package goyaad_test

import (
	"container/heap"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	uuid "github.com/satori/go.uuid"
	. "github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test jobs", func() {
	Context("Basic job tests", func() {
		It("can create a job", func() {
			b := []byte("foo")
			j := NewJob(uuid.NewV4().String(), time.Now(), b)
			Expect(j.IsReady()).To(BeTrue())
		})

		It("can create spoke bounds from job trigger time", func() {
			t := time.Unix(0, 0)
			j := NewJobAutoID(t.Add(time.Second*15), nil)

			sb := j.AsBound(time.Second)

			Expect(sb.Start().Before(j.TriggerAt()))
			Expect(sb.End().After(j.TriggerAt()))
		})
	})

	Context("Job Ordering", func() {
		It("orders jobs correctly", func() {
			t := time.Now()
			jone := NewJobAutoID(t.Add(1), nil)
			jtwo := NewJobAutoID(t.Add(20), nil)
			jthree := NewJobAutoID(t.Add(50), nil)
			ordList := []*Job{jone, jtwo, jthree}

			jobs := &PriorityQueue{jtwo.AsPriorityItem(), jone.AsPriorityItem(), jthree.AsPriorityItem()}
			heap.Init(jobs)

			for _, job := range ordList {
				j := heap.Pop(jobs).(*Item).Value().(*Job)
				Expect(j.ID()).To(Equal(job.ID()))
			}
		})
	})

	Context("Job serialization", func() {
		It("serde as gob", func() {
			j := NewJobAutoID(time.Now(), []byte("This is a test job"))
			encoded, err := j.GobEncode()
			Expect(err).To(BeNil())

			jj := &Job{}
			err = jj.GobDecode(encoded)
			Expect(err).To(BeNil())
		})

		It("use a persister to save a job", func() {
			j := NewJobAutoID(time.Now(), []byte("This is a test job"))
			persistenceTestDir := path.Join(os.TempDir(), "goyaadtest")
			p := persistence.NewLevelDBPersister(persistenceTestDir)
			errChan := p.Errors()
			Expect(p.ResetDataDir()).To(BeNil())

			err := j.Persist(p)
			Expect(err).NotTo(HaveOccurred())
			Eventually(errChan).ShouldNot(Receive())
		})
	})
})
