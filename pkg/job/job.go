package job

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/urjitbhatia/goyaad/internal/queue"
	"github.com/urjitbhatia/goyaad/internal/temporal"
)

// cache fixed size overhead
var sizeOverhead = uint64(unsafe.Sizeof(Job{}))

var serializationVersion = 1

// Job is the basic unit of work in yaad
type Job struct {
	id        string
	triggerAt time.Time
	body      []byte
}

// Impl Job

//NewJob creates a new yaad job
func NewJob(id string, triggerAt time.Time, b []byte) *Job {
	j := &Job{
		id:        id,
		triggerAt: triggerAt,
		body:      b,
	}
	return j
}

// NewJobAutoID creates a job a job id assigned automatically
func NewJobAutoID(triggerAt time.Time, b []byte) *Job {
	j := &Job{
		id:        fmt.Sprintf("%d", NextID()),
		triggerAt: triggerAt,
		body:      b,
	}
	return j
}

// SizeOf returns the memory allocated for this job in bytes
// including the size of the actual body payload + the fixed overhead costs
// Implements monitor.Sizeable interface
func (j *Job) SizeOf() uint64 {
	return sizeOverhead + uint64(len(j.body)+len(j.id))
}

// AsTemporalState returns the job's temporal classification at the point in time
func (j *Job) AsTemporalState() temporal.State {
	now := time.Now()
	switch {
	case now.After(j.triggerAt):
		return temporal.Past
	case j.triggerAt.After(now):
		return temporal.Future
	default:
		return temporal.Past
	}
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

// AsBound returns temporal.Bound for a hypothetical spoke that should hold this job
func (j *Job) AsBound(spokeSpan time.Duration) temporal.Bound {
	start := j.triggerAt.Truncate(spokeSpan)
	end := start.Add(spokeSpan)

	return temporal.NewBound(start, end)
}

// AsPriorityItem returns this job as a prioritizable item
func (j *Job) AsPriorityItem() *queue.Item {
	return queue.NewItem(j, j.triggerAt)
}

// GobEncode encodes a job into a binary buffer
func (j *Job) GobEncode() (data []byte, err error) {
	// We have a file for this Namespace
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)

	// insert serializationVersion first
	err = enc.Encode(serializationVersion)
	if err != nil {
		return nil, err
	}

	// id
	err = enc.Encode(j.id)
	if err != nil {
		return nil, err
	}
	//trigger at
	err = enc.Encode(j.triggerAt.UnixNano())
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
		log.Error().Err(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode encodes a job into a binary buffer
func (j *Job) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))

	// extract serializationVersion first
	var ver int
	err := dec.Decode(&ver)
	if err != nil {
		return err
	}
	// id
	err = dec.Decode(&j.id)
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
	//body
	err = dec.Decode(&j.body)
	if err == io.EOF {
		return nil
	}
	return err
}
