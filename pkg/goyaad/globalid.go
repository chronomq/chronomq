package goyaad

import (
	"sync/atomic"
)

var globalIDCtr uint64

// NextID generates the next int id monotonically increasing
func NextID() uint64 {
	return atomic.AddUint64(&globalIDCtr, 1)
}
