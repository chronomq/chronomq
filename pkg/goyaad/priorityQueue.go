package goyaad

import (
	"time"
)

// An Item is something we manage in a priority queue.
type Item struct {
	value    interface{} // The value of the item; Spoke or Job.
	priority time.Time   // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// Value pointed to by the item
func (i *Item) Value() interface{} {
	return i.value
}

// Priority of the item
func (i *Item) Priority() time.Time {
	return i.priority
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

// Len returns the current length of the underlying array of the PriorityQueue
func (pq PriorityQueue) Len() int { return len(pq) }

// Cap returns the current capacity of the underlying array of the Priority Queue
func (pq PriorityQueue) Cap() int { return cap(pq) }

// Less defines item ordering. Priority is defined by trigger time in the future
func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the item nearest in time, not highest.
	// if i starts AFTER j, i has lower priority
	return pq[i].priority.Before(pq[j].priority)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push an item to this PriorityQueue
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

// Pop the item with the closest trigger time (priority)
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old) - 1
	item := old[n]
	old = append(old[:n], old[n+1:]...)
	old[:n+1][n] = nil

	item.index = -1 // for safety

	*pq = old
	return item
}

// AtIdx gets item at given index
func (pq PriorityQueue) AtIdx(i int) *Item {
	return pq[i]
}
