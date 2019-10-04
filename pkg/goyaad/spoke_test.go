package goyaad_test

import (
	"container/heap"
	"math/rand"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"

	. "github.com/urjitbhatia/goyaad/pkg/goyaad"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

var _ = Describe("Test spokes", func() {
	Context("Basic spoke tests", func() {
		It("can create a spoke", func() {
			t := time.Now()

			// Starts in an hour, ends in 2 hours
			s := NewSpoke(t.Add(time.Hour*1), t.Add(time.Hour*2))
			Expect(s.IsReady()).To(BeFalse())
			Expect(s.IsExpired()).To(BeFalse())

			// Starts now, ends in 10 hours
			s = NewSpokeFromNow(time.Hour * 10)
			Expect(s.IsReady()).To(BeTrue())
			Expect(s.IsExpired()).To(BeFalse())

			// Ends in the past for the next tick
			s = NewSpokeFromNow(0)
			Expect(s.IsReady()).To(BeTrue())
			Expect(s.IsExpired()).To(BeTrue())
		})

		It("accepts jobs into spoke", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			j := NewJobAutoID(s.Start().Add(time.Second*20), nil)

			// Accepts the job and returns nil
			Expect(s.ContainsJob(j)).To(BeTrue())
			Expect(s.AddJobLocked(j)).To(BeNil())
			Expect(s.PendingJobsLen()).To(Equal(1))
			Expect(s.OwnsJobLocked(j.ID())).To(BeTrue())
		})

		It("rejects jobs from spoke that lie outside its time bounds", func() {
			now := time.Now()
			s := NewSpoke(now.Add(time.Hour*1), now.Add(time.Hour*2))

			// Job triggers after spoke ends
			j := NewJobAutoID(s.End().Add(time.Hour*1), nil)

			// Rejects the job and returns it
			Expect(s.AddJobLocked(j)).To(Not(BeNil()))
			Expect(s.PendingJobsLen()).To(Equal(0))

			// Job triggers before spoke ends
			j = NewJobAutoID(s.Start().Add(-10*time.Minute), nil)

			// Rejects the job and returns it
			Expect(s.AddJobLocked(j)).To(Not(BeNil()))
			Expect(s.PendingJobsLen()).To(Equal(0))
		})

		It("walks spoke with jobs", func() {
			s := NewSpokeFromNow(time.Hour * 1)

			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Nanosecond*time.Duration(rand.Intn(900))), nil)
				Expect(s.AddJobLocked(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(10))

			// Wait for all jobs to be ready
			time.Sleep(time.Second * 1)

			jobs := []*Job{}
			for s.PendingJobsLen() > 0 {
				jobs = append(jobs, s.NextLocked())
			}
			Expect(len(jobs)).To(Equal(10))
			prev := jobs[0]
			for i := 1; i < len(jobs); i++ {
				// Walk returns jobs in order
				Expect(prev.TriggerAt().Sub(jobs[i].TriggerAt()) <= 0).To(BeTrue())
			}
		})

		It("repeated walks spoke with jobs as they expire", func() {
			s := NewSpokeFromNow(time.Hour * 1)

			// Add some jobs < 1 sec triggerAt
			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Duration(rand.Intn(900))), nil)
				Expect(s.AddJobLocked(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(10))
			// Add some jobs > 1 sec triggerAt
			for i := 0; i < 10; i++ {
				j := NewJobAutoID(s.Start().Add(time.Second*20+time.Duration(10+rand.Intn(40))), nil)
				Expect(s.AddJobLocked(j)).To(BeNil())
			}
			Expect(s.PendingJobsLen()).To(Equal(20))

			// Wait for all jobs to be ready
			time.Sleep(time.Second)

			jobs := []*Job{}
			for s.PendingJobsLen() > 10 {
				j := s.NextLocked()
				jobs = append(jobs, j)
			}
			Expect(len(jobs)).To(Equal(10))
			prev := jobs[0]
			for i := 1; i < len(jobs); i++ {
				// Walk returns jobs in order
				Expect(prev.TriggerAt().Sub(jobs[i].TriggerAt()) <= 0).To(BeTrue())
			}

			// 10 jobs should remain
			Expect(s.PendingJobsLen()).To(Equal(10))
		})

		It("cancels job from spoke", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(s.PendingJobsLen()).To(Equal(0))

			j := NewJobAutoID(s.Start().Add(time.Minute*10), nil)
			Expect(s.AddJobLocked(j)).To(BeNil())
			Expect(s.PendingJobsLen()).To(Equal(1))

			s.CancelJobLocked(j.ID())
			Expect(s.PendingJobsLen()).To(Equal(0))
		})

		It("cancels job from spoke that doesnt exist does nothing", func() {
			s := NewSpokeFromNow(time.Hour * 1)
			Expect(s.PendingJobsLen()).To(Equal(0))
			s.CancelJobLocked(uuid.NewV4().String())
			s.CancelJobLocked(uuid.NewV4().String())
		})
	})

	Context("Spoke Ordering", func() {
		It("orders spokes correctly", func() {
			t := time.Now()
			sone := NewSpoke(t.Add(1), t.Add(10))
			stwo := NewSpoke(t.Add(20), t.Add(30))
			sthree := NewSpoke(t.Add(50), t.Add(55))
			ordList := []*Spoke{sone, stwo, sthree}

			spokes := &PriorityQueue{stwo.AsPriorityItem(), sone.AsPriorityItem(), sthree.AsPriorityItem()}
			heap.Init(spokes)

			// Expected order pop
			for _, spoke := range ordList {
				s := heap.Pop(spokes)
				item := s.(*Item)
				Expect(item.Value().(*Spoke).ID()).To(Equal(spoke.ID()))
			}
		})
	})
	Context("Spoke persistence", func() {
		It("persists a spoke", func() {
			s := NewSpokeFromNow(time.Minute * 100)
			persistenceTestDir := path.Join(os.TempDir(), "goyaadtest")
			p := persistence.NewJournalPersister(persistenceTestDir)
			Expect(p.ResetDataDir()).To(BeNil())

			errC := s.PersistLocked(p)
			Eventually(errC).ShouldNot(Receive())
		})
	})
})
