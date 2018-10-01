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
	jobMap   map[uuid.UUID]*Job
	jobQueue JobsByTime
}

// spokeBound defines time bounds for a spoke
type spokeBound struct {
	start time.Time
	end   time.Time
}

// -- SpokeBound -- //
// Start returns the starting time of this spoke bound (inclusive)
func (sb *spokeBound) Start() time.Time {
	return sb.start
}

// End returns the ending time of this spoke bound (exclusive)
func (sb *spokeBound) End() time.Time {
	return sb.end
}

// Contains returns true if sb fully contains o
func (sb *spokeBound) Contains(o *spokeBound) bool {
	return sb.start.Sub(o.start) <= 0 && sb.end.After(o.end)
}

// ContainsJob returns true if this job is bounded by this spoke
func (sb *spokeBound) ContainsJob(j *Job) bool {
	return sb.start.Sub(j.triggerAt) <= 0 && sb.end.After(j.triggerAt)
}

// IsReady returns true if SpokeBound started in the past
// This spoke bound may end in the future
func (sb *spokeBound) IsReady() bool {
	return time.Now().After(sb.start)
}

// IsExpired returns true if SpokeBound ended in the past
func (sb *spokeBound) IsExpired() bool {
	return time.Now().After(sb.end)
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
		jobMap:     make(map[uuid.UUID]*Job),
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
		logrus.WithFields(logrus.Fields{"job": j,
			"spoke": s}).Debug("Accepting job")
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

	if len(s.jobQueue) == 0 {
		return &ready
	}

	sort.Sort(s.jobQueue)
	var j *Job
	for {
		if len(s.jobQueue) == 0 {
			break
		}

		if time.Now().Before(s.jobQueue[0].triggerAt) {
			break
		}

		j, s.jobQueue = s.jobQueue[0], s.jobQueue[1:]
		delete(s.jobMap, j.id)
		ready = append(ready, j)

	}
	return &ready
}

// CancelJob will try to delete a job that hasn't been consumed yet
func (s *Spoke) CancelJob(id uuid.UUID) {
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
func (s *Spoke) OwnsJob(id uuid.UUID) bool {
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
