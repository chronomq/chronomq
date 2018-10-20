package goyaad

import (
	"fmt"
	"time"

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
		id:        fmt.Sprintf("%d", NextID()),
		triggerAt: triggerAt,
		body:      b,
	}
}

// AsTemporalState returns the job's temporal classification at the point in time
func (j *Job) AsTemporalState() TemporalState {
	now := time.Now()
	switch {
	case now.After(j.triggerAt):
		return Past
	case j.triggerAt.After(now):
		return Future
	default:
		return Past
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

// AsPriorityItem returns this job as a prioritizable item
func (j *Job) AsPriorityItem() *Item {
	return &Item{index: 0, priority: j.triggerAt, value: j}
}
