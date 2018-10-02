package goyaad_test

import (
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	uuid "github.com/satori/go.uuid"
	. "github.com/urjitbhatia/goyaad/pkg/yaad"
)

var _ = Describe("Test jobs", func() {
	Context("Basic job tests", func() {
		It("can create a job", func() {
			b := []byte("foo")
			_, err := NewJob(uuid.NewV4(), time.Now(), &b)
			Expect(err).To(BeNil())

			_, err = NewJob(uuid.NewV4(), time.Now(), nil)
			Expect(err).To(BeNil())

			j, err := NewJobAutoID(time.Now(), nil)
			Expect(err).To(BeNil())
			Expect(j.ID()).To(BeAssignableToTypeOf(uuid.NewV4()))

			Expect(j.IsReady()).To(BeTrue())
		})

		It("can create spoke bounds from job trigger time", func() {
			t := time.Unix(0, 0)
			j, err := NewJobAutoID(t.Add(time.Second*15), nil)
			Expect(err).To(BeNil())
			Expect(j.ID()).To(BeAssignableToTypeOf(uuid.NewV4()))

			sb := j.AsBound(time.Second)

			Expect(sb.Start().Before(j.TriggerAt()))
			Expect(sb.End().After(j.TriggerAt()))
		})
	})

	Context("Job Ordering", func() {
		It("orders jobs correctly", func() {
			t := time.Now()
			jone, _ := NewJobAutoID(t.Add(1), nil)
			jtwo, _ := NewJobAutoID(t.Add(20), nil)
			jthree, _ := NewJobAutoID(t.Add(50), nil)

			jobs := JobsByTime{jtwo, jone, jthree}
			sort.Sort(jobs)

			Expect(jobs[0].ID()).To(Equal(jone.ID()))
			Expect(jobs[1].ID()).To(Equal(jtwo.ID()))
			Expect(jobs[2].ID()).To(Equal(jthree.ID()))
		})
	})
})
