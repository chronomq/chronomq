package goyaad

import "sync"

var globalIDCtr = 0
var idMutex = sync.Mutex{}

// NextID generates the next int id monotonically increasing
func NextID() int {
	idMutex.Lock()
	defer idMutex.Unlock()

	globalIDCtr++
	return globalIDCtr
}
