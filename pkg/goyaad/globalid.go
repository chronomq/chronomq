package goyaad

import (
	"sync"
	"sync/atomic"
)

var globalIDCtr uint64
var idMutex = sync.Mutex{}

// NextID generates the next int id monotonically increasing
func NextID() uint64 {
	return atomic.AddUint64(&globalIDCtr, 1)
}
