package yaad

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"

	"github.com/urjitbhatia/yaad/pkg/persistence"
)

// Spoke is a time bound chain of jobs
type Spoke struct {
	id uuid.UUID
	SpokeBound
	jobMap   map[string]bool // Provides quicker lookup of jobs owned by this spoke
	jobQueue PriorityQueue   // Orders the jobs by trigger priority

	lock *sync.Mutex
}

// ErrJobOutOfSpokeBounds is returned when an attempt was made to add a job to a spoke that
// should not contain it - the job's trigger time it outside the spoke bounds
var ErrJobOutOfSpokeBounds = errors.New("The offered job is outside the bounds of this spoke ")

// -- Spoke -- //

// NewSpokeFromNow creates a new spoke to hold jobs that starts from now
// and ends at the given duration
func NewSpokeFromNow(duration time.Duration) *Spoke {
	now := time.Now()
	return NewSpoke(now, now.Add(duration))
}

// NewSpoke creates a new spoke to hold jobs
func NewSpoke(start, end time.Time) *Spoke {
	jq := PriorityQueue{}
	heap.Init(&jq)
	return &Spoke{id: uuid.NewV4(),
		jobMap:     make(map[string]bool),
		jobQueue:   jq,
		SpokeBound: SpokeBound{start, end},
		lock:       &sync.Mutex{}}
}

// GetLocker returns the spoke as a sync.Locker interface
func (s *Spoke) GetLocker() sync.Locker {
	return s
}

// Lock this spoke
func (s *Spoke) Lock() {
	s.lock.Lock()
}

// Unlock this spoke
func (s *Spoke) Unlock() {
	s.lock.Unlock()
}

// AsTemporalState returns the spoke's temporal classification at the point in time
func (s *Spoke) AsTemporalState() TemporalState {
	now := time.Now()
	switch {
	case s.end.Before(now):
		return Past
	case s.start.After(now):
		return Future
	default:
		return Current
	}
}

// AddJobLocked submits a job to the spoke. If the spoke cannot take responsibility
// of this job, it will return it as it is, otherwise nil is returned
func (s *Spoke) AddJobLocked(j *Job) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.ContainsJob(j) {
		return ErrJobOutOfSpokeBounds
	}
	log.Debug().
		Str("jobID", j.id).
		Time("triggerAt", j.triggerAt).
		Str("spokeID", s.id.String()).
		Time("spokeStart", s.start).
		Time("spokeEnd", s.end).
		Msg("Accepting new job")

	s.jobMap[j.id] = true
	heap.Push(&s.jobQueue, j.AsPriorityItem())
	return nil
}

// NextLocked returns the next ready job
func (s *Spoke) NextLocked() *Job {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.jobQueue.Len() == 0 {
		return nil
	}
	// Peek
	i := s.jobQueue.AtIdx(0)
	if i == nil {
		return nil
	}

	j := i.value.(*Job)
	switch j.AsTemporalState() {
	case Past, Current:
		// pop from queue
		delete(s.jobMap, j.id)
		heap.Pop(&s.jobQueue)
		return j
	default:
		return nil
	}
}

// CancelJobLocked will try to delete a job that hasn't been consumed yet
func (s *Spoke) CancelJobLocked(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.jobMap[id]; ok {
		delete(s.jobMap, id)
		// Also delete from pq
		for i, j := range s.jobQueue {
			if j.value.(*Job).id == id {
				heap.Remove(&s.jobQueue, i)
				return nil
			}
		}
	}
	return fmt.Errorf("Cannot find job to cancel")
}

// OwnsJobLocked returns true if a job by given id is owned by this spoke
func (s *Spoke) OwnsJobLocked(id string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.jobMap[id]
	return ok
}

// PendingJobsLen returns the number of jobs remaining in this spoke
func (s *Spoke) PendingJobsLen() int {
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

// PersistLocked all jobs in this spoke
func (s *Spoke) PersistLocked(p persistence.Persister) chan error {
	s.Lock()
	errC := make(chan error)
	go func() {
		defer close(errC)
		defer s.Unlock()
		var i = 0
		for i = 0; i < s.jobQueue.Len(); i++ {
			err := p.Persist(s.jobQueue.AtIdx(i).Value().(*Job))
			if err != nil {
				errC <- err
				continue
			}
		}
		log.Info().
			Int("jobCount", i).
			Str("spokeID", s.id.String()).
			Msg("Persisted jobs from spoke")
	}()
	return errC
}
