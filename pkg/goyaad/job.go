package goyaad

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Job is the basic unit of work in yaad
type Job struct {
	id        string
	triggerAt time.Time
	body      []byte

	pri int32
	ttr time.Duration
}

// Impl Job

//NewJob creates a new yaad job
func NewJob(id string, triggerAt time.Time, b []byte) *Job {
	return &Job{
		id:        id,
		triggerAt: triggerAt,
		body:      b,
	}
}

// NewJobAutoID creates a job a job id assigned automatically
func NewJobAutoID(triggerAt time.Time, b []byte) *Job {
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
func (j *Job) Body() []byte {
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
func (j *Job) AsBound(spokeSpan time.Duration) SpokeBound {
	start := j.triggerAt.Truncate(spokeSpan)
	end := start.Add(spokeSpan)

	return SpokeBound{start: start, end: end}
}

// AsPriorityItem returns this job as a prioritizable item
func (j *Job) AsPriorityItem() *Item {
	return &Item{index: 0, priority: j.triggerAt, value: j}
}

// GobEncode encodes a job into a binary buffer
func (j *Job) GobEncode() (data []byte, err error) {
	// We have a file for this Namespace
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	// id
	err = enc.Encode(j.id)
	if err != nil {
		return nil, err
	}
	//pri
	err = enc.Encode(j.pri)
	if err != nil {
		return nil, err
	}
	//trigger at
	err = enc.Encode(j.triggerAt.UnixNano())
	if err != nil {
		return nil, err
	}
	//ttr
	err = enc.Encode(j.ttr)
	if err != nil {
		return nil, err
	}
	//body
	err = enc.Encode(j.body)
	if err != nil {
		return nil, err
	}

	if err != nil {
		err = errors.Wrap(err, "Job: Failed to encode job for persistence")
		logrus.Error(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode encodes a job into a binary buffer
func (j *Job) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))

	// id
	err := dec.Decode(&j.id)
	if err != nil {
		return err
	}
	// pri
	err = dec.Decode(&j.pri)
	if err != nil {
		return err
	}
	//trigger at
	var triggerAtUnixNano int64
	err = dec.Decode(&triggerAtUnixNano)
	if err != nil {
		return err
	}
	j.triggerAt = time.Unix(0, triggerAtUnixNano)
	//ttr
	err = dec.Decode(&j.ttr)
	if err != nil {
		return err
	}
	//body
	err = dec.Decode(&j.body)
	if err == io.EOF {
		return nil
	}
	return err
}
