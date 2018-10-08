package goyaad

import (
	"sort"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Spoke is a time bound chain of jobs
type Spoke struct {
	id uuid.UUID
	*spokeBound
	jobMap   map[string]*Job
	jobQueue JobsByTime
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
	return &Spoke{id: uuid.NewV4(),
		jobMap:     make(map[string]*Job),
		jobQueue:   JobsByTime{},
		spokeBound: &spokeBound{start, end}}
}

// AddJob submits a job to the spoke. If the spoke cannot take responsibility
// of this job, it will return it as it is, otherwise nil is returned
func (s *Spoke) AddJob(j *Job) *Job {
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
		s.jobMap[j.id] = j
		s.jobQueue = append(s.jobQueue, j)
		return nil
	}

	return j
}

// Walk returns a pointer to an array of pointers to Jobs
// (no copy operations)
func (s *Spoke) Walk() *[]*Job {
	ready := []*Job{}

	j := s.Next()
	for j != nil {
		ready = append(ready, j)
		j = s.Next()
	}
	return &ready
}

// Next returns the next ready job
func (s *Spoke) Next() *Job {
	if len(s.jobQueue) == 0 {
		return nil
	}
	sort.Sort(s.jobQueue)
	if time.Now().After(s.jobQueue[0].triggerAt) {
		var j *Job
		j, s.jobQueue = s.jobQueue[0], s.jobQueue[1:]
		delete(s.jobMap, j.id)
		return j
	}
	return nil
}

// CancelJob will try to delete a job that hasn't been consumed yet
func (s *Spoke) CancelJob(id string) {
	if _, ok := s.jobMap[id]; ok {
		delete(s.jobMap, id)
		for i, j := range s.jobQueue {
			if j.id == id {
				s.jobQueue = append(s.jobQueue[:i], s.jobQueue[i+1:]...)
				break
			}
		}
	}
}

// OwnsJob returns true if a job by given id is owned by this spoke
func (s *Spoke) OwnsJob(id string) bool {
	_, ok := s.jobMap[id]
	return ok
}

// PendingJobsLen returns the number of jobs remaining in this spoke
func (s *Spoke) PendingJobsLen() int {
	return len(s.jobMap)
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
