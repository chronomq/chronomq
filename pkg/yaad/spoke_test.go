package goyaad_test

import (
	"math/rand"
	"sort"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	uuid "github.com/satori/go.uuid"
	. "github.com/urjitbhatia/goyaad/pkg/yaad"
)

var _ = Describe("Test spokes", func() {
	Context("Basic spoke tests", func() {
		It("can create a spoke", func() {
			t := time.Now()

			// Starts in an hour, ends in 2 hours
			s := NewSpoke(t.Add(time.Hour*1), t.Add(time.Hour*2))
			Expect(s.ID()).To(BeAssignableToTypeOf(uuid.NewV4()))
			Expect(s.IsReady()).To(BeFalse())
			Expect(s.IsExpired()).To(BeFalse())

			// Starts now, ends in 10 hours
			s = NewSpokeFromNow(time.Hour * 10)
			Expect(s.ID()).To(BeAssignableToTypeOf(uuid.NewV4()))
			Expect(s.IsReady()).To(BeTrue())
			Expect(s.IsExpired()).To(BeFalse())

			// Ends in the past for the next tick
			s = NewSpokeFromNow(0)
			Expect(s.ID()).To(BeAssignableToTypeOf(uuid.NewV4()))
			Expect(s.IsReady()).To(BeTrue())
			Expect(s.IsExpired()).To(BeTrue())
		})

		It("accepts jobs into spoke", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			j := NewJobAutoID(s.Start().Add(time.Second*20), nil)

			// Accepts the job and returns nil
			Expect(s.ContainsJob(j)).To(BeTrue())
			Expect(s.AddJob(j)).To(BeNil())
			Expect(s.PendingJobsLen()).To(Equal(1))
			Expect(s.OwnsJob(j.ID())).To(BeTrue())
		})

		It("rejects jobs from spoke that lie outside its time bounds", func() {
			now := time.Now()
			s := NewSpoke(now.Add(time.Hour*1), now.Add(time.Hour*2))

			// Job triggers after spoke ends
			j := NewJobAutoID(s.End().Add(time.Hour*1), nil)

			// Rejects the job and returns it
			Expect(s.AddJob(j)).To(Equal(j))
			Expect(s.PendingJobsLen()).To(Equal(0))

			// Job triggers before spoke ends
			j = NewJobAutoID(s.Start().Add(-10*time.Minute), nil)

			// Rejects the job and returns it
			Expect(s.AddJob(j)).To(Equal(j))
			Expect(s.PendingJobsLen()).To(Equal(0))
		})

		It("walks empty spoke", func() {
			s := NewSpokeFromNow(time.Minute * 1)
			Expect(len(*s.Walk())).To(Equal(0))
		})

		It("walks spoke with jobs", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(len(*s.Walk())).To(Equal(0))

			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Nanosecond*time.Duration(rand.Intn(900))), nil)
				Expect(s.AddJob(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(10))

			// Wait for all jobs to be ready
			time.Sleep(time.Second * 1)

			jobs := *s.Walk()
			Expect(len(jobs)).To(Equal(10))
			prev := jobs[0]
			for i := 1; i < len(jobs); i++ {
				// Walk returns jobs in order
				Expect(prev.TriggerAt().Sub(jobs[i].TriggerAt()) <= 0).To(BeTrue())
			}
		})

		It("repeated walks spoke with jobs as they expire", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(len(*s.Walk())).To(Equal(0))

			// Add some jobs < 1 sec triggerAt
			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Nanosecond*time.Duration(rand.Intn(900))), nil)
				Expect(s.AddJob(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(10))
			// Add some jobs > 1 sec triggerAt
			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Minute*time.Duration(10+rand.Intn(40))), nil)
				Expect(s.AddJob(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(20))

			// Wait for all jobs to be ready
			time.Sleep(time.Second * 1)

			jobs := *s.Walk()
			Expect(len(jobs)).To(Equal(10))
			prev := jobs[0]
			for i := 1; i < len(jobs); i++ {
				// Walk returns jobs in order
				Expect(prev.TriggerAt().Sub(jobs[i].TriggerAt()) <= 0).To(BeTrue())
			}

			// 10 jobs should remain
			Expect(s.PendingJobsLen()).To(Equal(10))

			// Walk is idempotent
			jobs = *s.Walk()
			Expect(len(jobs)).To(Equal(0))
			Expect(s.PendingJobsLen()).To(Equal(10))
		})

		It("cancels job from spoke", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(s.PendingJobsLen()).To(Equal(0))

			j := NewJobAutoID(s.Start().Add(time.Minute*10), nil)
			Expect(s.AddJob(j)).To(BeNil())
			Expect(s.PendingJobsLen()).To(Equal(1))

			s.CancelJob(j.ID())
			Expect(s.PendingJobsLen()).To(Equal(0))
		})

		It("cancels job from spoke that doesnt exist does nothing", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(s.PendingJobsLen()).To(Equal(0))
			s.CancelJob(uuid.NewV4().String())
			s.CancelJob(uuid.NewV4().String())
		})
	})

	Context("Spoke Ordering", func() {
		It("orders spokes correctly", func() {
			t := time.Now()
			sone := NewSpoke(t.Add(1), t.Add(10))
			stwo := NewSpoke(t.Add(20), t.Add(30))
			sthree := NewSpoke(t.Add(50), t.Add(55))

			spokes := SpokesByTime{stwo, sone, sthree}
			sort.Sort(spokes)

			Expect(spokes[0].ID()).To(Equal(sone.ID()))
			Expect(spokes[1].ID()).To(Equal(stwo.ID()))
			Expect(spokes[2].ID()).To(Equal(sthree.ID()))
		})
	})
})
