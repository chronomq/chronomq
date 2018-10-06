package goyaad

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// Job is the basic unit of work in yaad
type Job struct {
	id        string
	triggerAt time.Time
	body      *[]byte

	pri int32
	ttr time.Duration
}

// Impl Job

//NewJob creates a new yaad job
func NewJob(id string, triggerAt time.Time, b *[]byte) *Job {
	return &Job{
		id:        id,
		triggerAt: triggerAt,
		body:      b,
	}
}

// NewJobAutoID creates a job a job id assigned automatically
func NewJobAutoID(triggerAt time.Time, b *[]byte) *Job {
	return &Job{
		id:        uuid.NewV4().String(),
		triggerAt: triggerAt,
		body:      b,
	}
}

// SetOpts sets job options
func (j *Job) SetOpts(pri int32, ttr time.Duration) {
	j.pri = pri
	j.ttr = ttr
}

// ID returns the id of the job
func (j *Job) ID() string {
	return j.id
}

// Body returns the job of the job
func (j *Job) Body() *[]byte {
	return j.body
}

// TriggerAt returns the job's trigger time
func (j *Job) TriggerAt() time.Time {
	return j.triggerAt
}

// IsReady returns true if job is ready to be worked on
func (j *Job) IsReady() bool {
	return time.Now().After(j.triggerAt)
}

// AsBound returns spokeBound for a hypothetical spoke that should hold this job
func (j *Job) AsBound(spokeSpan time.Duration) spokeBound {
	start := j.triggerAt.Truncate(spokeSpan)
	logrus.Debugf("Start floor unixnano: %+v", start.UnixNano())

	end := start.Add(spokeSpan)
	logrus.Debugf("End unixnano: %+v", end.UnixNano())

	return spokeBound{start: start, end: end}
}

// JobsByTime implements sort.Interface for a collection of jobs //
type JobsByTime []*Job

// Len is the number of elements in the collection.
func (b JobsByTime) Len() int {
	return len(b)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (b JobsByTime) Less(i, j int) bool {
	return b[i].triggerAt.Before(b[j].triggerAt)
}

// Swap swaps the elements with indexes i and j.
func (b JobsByTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
