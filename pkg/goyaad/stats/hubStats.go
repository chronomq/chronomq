package stats

import "sync/atomic"

// Counters are used to keep quick stats counters so that we can do
// stats reporting without locking the hub and reading over the spokes
// internal state should not be accessed directly
type Counters struct {
	s Snapshot
}

// Snapshot is a readonly view of the stats counters
type Snapshot struct {
	CurrentJobs int64 // current set of jobs
	RemovedJobs int64 // jobs removed so far
}

// Read returns a "copy" of the current stats snapshot at that instant
func (c *Counters) Read() Snapshot {
	r := Snapshot{}
	r.CurrentJobs = atomic.LoadInt64(&c.s.CurrentJobs)
	r.RemovedJobs = atomic.LoadInt64(&c.s.RemovedJobs)
	return r
}

// IncrJob updates counters - job has been added
func (c *Counters) IncrJob() {
	atomic.AddInt64(&c.s.CurrentJobs, 1)
}

// DecrJob updates counter - job has been removed
func (c *Counters) DecrJob() {
	atomic.AddInt64(&c.s.CurrentJobs, -1)
	atomic.AddInt64(&c.s.RemovedJobs, 1)
}
