package goyaad

import "time"

// SpokeBound defines time bounds for a spoke
type SpokeBound struct {
	start time.Time
	end   time.Time
}

// Start returns the starting time of this spoke bound (inclusive)
func (sb *SpokeBound) Start() time.Time {
	return sb.start
}

// End returns the ending time of this spoke bound (exclusive)
func (sb *SpokeBound) End() time.Time {
	return sb.end
}

// Contains returns true if sb fully contains o
func (sb *SpokeBound) Contains(o *SpokeBound) bool {
	return sb.start.Sub(o.start) <= 0 && sb.end.After(o.end)
}

// ContainsJob returns true if this job is bounded by this spoke
func (sb *SpokeBound) ContainsJob(j *Job) bool {
	return sb.start.Sub(j.triggerAt) <= 0 && sb.end.After(j.triggerAt)
}

// IsReady returns true if SpokeBound started in the past
// This spoke bound may end in the future
func (sb *SpokeBound) IsReady() bool {
	return time.Now().After(sb.start)
}

// IsExpired returns true if SpokeBound ended in the past
func (sb *SpokeBound) IsExpired() bool {
	return time.Now().After(sb.end)
}
