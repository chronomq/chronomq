package goyaad

import (
	"errors"
	"time"

	uuid "github.com/satori/go.uuid"
)

// Job is the basic unit of work in yaad
type Job struct {
	id        uuid.UUID
	triggerAt time.Time
	body      *[]byte
}

// Impl Job

//NewJob creates a new yaad job
func NewJob(id uuid.UUID, triggerAt time.Time, b *[]byte) (*Job, error) {
	switch id.Version() {
	case 4:
		return &Job{
			id:        id,
			triggerAt: triggerAt,
			body:      b,
		}, nil
	default:
		return nil, errors.New("Only uuid v4 ids are accepted")
	}
}

// NewJobAutoID creates a job a job id assigned automatically
func NewJobAutoID(triggerAt time.Time, b *[]byte) (*Job, error) {
	return &Job{
		id:        uuid.NewV4(),
		triggerAt: triggerAt,
		body:      b,
	}, nil
}

// ID returns the id of the job
func (j *Job) ID() uuid.UUID {
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
