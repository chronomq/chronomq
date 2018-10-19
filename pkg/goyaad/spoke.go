package goyaad

import (
	"container/heap"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Spoke is a time bound chain of jobs
type Spoke struct {
	id uuid.UUID
	*spokeBound
	jobMap   sync.Map
	jobQueue *PriorityQueue

	lock sync.Mutex
}

// -- Spoke -- //

// NewSpokeFromNow creates a new spoke to hold jobs that starts from now
// and ends at the given duration
func NewSpokeFromNow(duration time.Duration) *Spoke {
	now := time.Now()
	return NewSpoke(now, now.Add(duration))
}

// NewSpoke creates a new spoke to hold jobs
func NewSpoke(start, end time.Time) *Spoke {
	jq := &PriorityQueue{}
	heap.Init(jq)
	return &Spoke{id: uuid.NewV4(),
		jobMap:     sync.Map{},
		jobQueue:   jq,
		spokeBound: &spokeBound{start, end},
		lock:       sync.Mutex{}}
}

// AddJob submits a job to the spoke. If the spoke cannot take responsibility
// of this job, it will return it as it is, otherwise nil is returned
func (s *Spoke) AddJob(j *Job) *Job {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.IsExpired() {
		return j
	}

	if s.ContainsJob(j) {
		logrus.WithFields(
			logrus.Fields{
				"jobID":        j.id,
				"jobTriggerAt": j.triggerAt.UnixNano(),
				"spokeID":      s.id,
				"spokeStart":   s.start.UnixNano(),
				"spokeEnd":     s.end.UnixNano(),
			}).Debug("Accepting job")
		s.jobMap.Store(j.id, j)
		heap.Push(s.jobQueue, j.AsPriorityItem())
		return nil
	}

	return j
}

// Next returns the next ready job
func (s *Spoke) Next() *Job {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.jobQueue.Len() == 0 {
		return nil
	}
	i := heap.Pop(s.jobQueue)
	if i != nil {
		j := i.(*Item).value.(*Job)
		if j.triggerAt.Before(time.Now()) {
			return j
		}
		// No need to continue looking if the smallest item is still in the future
		heap.Push(s.jobQueue, i)
	}
	return nil
}

// CancelJob will try to delete a job that hasn't been consumed yet
func (s *Spoke) CancelJob(id string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.jobMap.Load(id); ok {
		s.jobMap.Delete(id)
		for i, j := range *s.jobQueue {
			if j.value.(*Job).id == id {
				heap.Remove(s.jobQueue, i)
				break
			}
		}
	}
}

// OwnsJob returns true if a job by given id is owned by this spoke
func (s *Spoke) OwnsJob(id string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.jobMap.Load(id)
	return ok
}

// PendingJobsLen returns the number of jobs remaining in this spoke
func (s *Spoke) PendingJobsLen() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.jobQueue.Len()
}

// ID returns the id of this spoke
func (s *Spoke) ID() uuid.UUID {
	return s.id
}

// AsPriorityItem returns a spoke as a prioritizable Item
func (s *Spoke) AsPriorityItem() *Item {
	return &Item{index: 0, priority: s.start, value: s}
}

// -- Implement Sort.Interface on Spokes ---//

// SpokesByTime implements sort.Interface for Spokes
type SpokesByTime []*Spoke

func (b SpokesByTime) Len() int {
	return len(b)
}

func (b SpokesByTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b SpokesByTime) Less(i, j int) bool {
	// i starts before or at the same time as j
	return b[i].start.Sub(b[j].start) <= 0 &&
		// && i ends before or at the same time as j
		b[i].end.Sub(b[j].end) <= 0
}
