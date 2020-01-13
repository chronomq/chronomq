package temporal

import (
	"time"
)

// Bound defines a time interval
type Bound struct {
	start time.Time
	end   time.Time
}

// NewBound creates a spoke-bound within start,end time bounds
func NewBound(start, end time.Time) Bound {
	return Bound{start, end}
}

// Start returns the starting time of this spoke bound (inclusive)
func (sb *Bound) Start() time.Time {
	return sb.start
}

// End returns the ending time of this spoke bound (exclusive)
func (sb *Bound) End() time.Time {
	return sb.end
}

// Contains returns true if sb fully contains o
func (sb *Bound) Contains(o *Bound) bool {
	return sb.start.Sub(o.start) <= 0 && sb.end.After(o.end)
}

// ContainsTime returns true if this time instant is bounded by this spoke
func (sb *Bound) ContainsTime(t time.Time) bool {
	return sb.start.Sub(t) <= 0 && sb.end.After(t)
}

// IsStarted returns true if this time Bound started in the past
// time.Now() may or may not be after this bound's end
func (sb *Bound) IsStarted() bool {
	return time.Now().After(sb.start)
}

// IsExpired returns true if SpokeBound ended in the past
func (sb *Bound) IsExpired() bool {
	return time.Now().After(sb.end)
}
