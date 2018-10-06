package goyaad_test

import (
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	uuid "github.com/satori/go.uuid"
	. "github.com/urjitbhatia/goyaad/pkg/goyaad"
)

var _ = Describe("Test jobs", func() {
	Context("Basic job tests", func() {
		It("can create a job", func() {
			b := []byte("foo")
			j := NewJob(uuid.NewV4().String(), time.Now(), &b)
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

			jobs := JobsByTime{jtwo, jone, jthree}
			sort.Sort(jobs)

			Expect(jobs[0].ID()).To(Equal(jone.ID()))
			Expect(jobs[1].ID()).To(Equal(jtwo.ID()))
			Expect(jobs[2].ID()).To(Equal(jthree.ID()))
		})
	})
})
