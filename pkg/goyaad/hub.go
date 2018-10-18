package goyaad

import (
	"container/heap"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	hundredYears = time.Hour * 24 * 365 * 100
)

// Hub is a time ordered collection of spokes
type Hub struct {
	spokeSpan    time.Duration
	spokeMap     map[*spokeBound]*Spoke // quick lookup map
	spokes       *PriorityQueue
	pastSpoke    *Spoke
	reservedJobs map[string]*Job // This could also be a spoke that order by TTL - optimize later

	removedJobsCount uint64
	lock             *sync.Mutex
}

// NewHub creates a new hub where adjacent spokes lie at the given
// spokeSpan duration boundary.
func NewHub(spokeSpan time.Duration) *Hub {
	h := &Hub{
		spokeSpan,
		make(map[*spokeBound]*Spoke),
		&PriorityQueue{},
		// Spoke from -100 years to 100 years
		NewSpoke(time.Now().Add(-1*hundredYears), time.Now().Add(hundredYears)),
		make(map[string]*Job),
		0,
		&sync.Mutex{},
	}
	heap.Init(h.spokes)
	go h.pruner()

	logrus.WithFields(logrus.Fields{
		"start": h.pastSpoke.start.String(),
		"end":   h.pastSpoke.end.String(),
	}).Debug("Created hub with past spoke")

	return h
}

func (h *Hub) pruner() {
	pruneT := time.NewTicker(time.Second * 5)
	statusT := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-pruneT.C:
			h.lock.Lock()
			for k, v := range h.reservedJobs {
				if time.Now().Sub(v.triggerAt) < v.ttr {
					delete(h.reservedJobs, k)
					h.AddJob(v)
				}
			}
			// Also prune empty spokes
			h.Prune()
			h.lock.Unlock()
		case <-statusT.C:
			h.Status()
		}
	}
}

// PendingJobsCount return the number of jobs currently pending
func (h *Hub) PendingJobsCount() int {
	count := h.pastSpoke.PendingJobsLen()
	for _, v := range h.spokeMap {
		count += v.PendingJobsLen()
	}

	return count
}

// CancelJob cancels a job if found. Calls are noop for unknown jobs
func (h *Hub) CancelJob(jobID string) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Search if this job is reserved
	if _, ok := h.reservedJobs[jobID]; ok {
		delete(h.reservedJobs, jobID)
		h.removedJobsCount++
		return nil
	}
	s, err := h.FindOwnerSpoke(jobID)
	if err != nil {
		return err
	}

	s.CancelJob(jobID)
	h.removedJobsCount++
	return nil
}

// FindOwnerSpoke returns the spoke that owns this job
func (h *Hub) FindOwnerSpoke(jobID string) (*Spoke, error) {
	if h.pastSpoke.OwnsJob(jobID) {
		return h.pastSpoke, nil
	}

	for _, v := range h.spokeMap {
		if v.OwnsJob(jobID) {
			return v, nil
		}
	}
	return nil, errors.New("Cannot find job owner spoke")
}

// AddSpoke adds spoke s to this hub
func (h *Hub) AddSpoke(s *Spoke) {
	h.spokeMap[s.spokeBound] = s
	heap.Push(h.spokes, s.AsPriorityItem())
}

// Walk returns a Vector of Jobs that should be consumed next
func (h *Hub) Walk() *[]*Job {
	ready := []*Job{}

	// collect jobs from past spoke
	ready = append(ready, *h.pastSpoke.Walk()...)
	logrus.Debugf("Got %d jobs from past spoke", len(ready))

	logrus.Debug("queries all spokes")

	for j := h.Next(); j != nil; j = h.Next() {
		ready = append(ready, j)
	}

	return &ready
}

// Next returns the next job that is ready now or returns nil.
func (h *Hub) Next() *Job {
	h.lock.Lock()
	defer h.lock.Unlock()

	j := h.pastSpoke.Next()
	if j != nil {
		logrus.Debug("Got job from past spoke")
		return j
	}

	pq := &PriorityQueue{}
	heap.Init(pq)
	defer h.mergeQueues(pq)

	logrus.Debug("queries all spokes")
	for h.spokes.Len() > 0 {
		// iterate spokes in order
		i := heap.Pop(h.spokes).(*Item)
		// save it in our temp pq
		heap.Push(pq, i)

		// extract read job from this spoke
		s := i.value.(*Spoke)
		j = s.Next()
		if j != nil {
			h.reserve(j)
			return j
		}
	}

	return nil
}

func (h *Hub) reserve(j *Job) {
	h.reservedJobs[j.id] = j
}

func (h *Hub) mergeQueues(pq *PriorityQueue) {
	for pq.Len() > 0 {
		i := heap.Pop(pq)
		h.spokes.Push(i)
	}
}

// Prune clears spokes which are expired and have no jobs
// returns the number of spokes pruned
func (h *Hub) Prune() int {
	pruned := 0
	for k, v := range h.spokeMap {
		if v.IsExpired() && v.PendingJobsLen() == 0 {
			delete(h.spokeMap, k)
		}
		pruned++
	}

	return pruned
}

// AddJob to this hub. Hub should never reject a job - this method will panic if that happens
func (h *Hub) AddJob(j *Job) *Hub {

	logrus.WithField("TriggerAt", j.triggerAt.UnixNano()).Debug("Adding job to hub.")
	if !h.maybeAddToPast(j) {
		h.addToSpokes(j)
	}
	return h
}

func (h *Hub) maybeAddToPast(j *Job) bool {
	if j.triggerAt.Before(time.Now()) {
		logrus.WithField("JobID", j.ID).Debug("Adding job to past spoke")
		rejected := h.pastSpoke.AddJob(j)
		if rejected != nil {
			logrus.Error("Past spoke rejected job. This should never happen")
		}

		return true
	}

	// rejected job
	return false
}

func (h *Hub) addToSpokesFast(j *Job) {
	// Traverse in order
	acceped := false

	// heap.Init(h.spokes)

	scanned := 0
	for j != nil && scanned < h.spokes.Len() {
		s := h.spokes.AtIdx(scanned).value.(*Spoke)
		j = s.AddJob(j)
		if j == nil {
			acceped = true
		}
		scanned++
	}

	if !acceped {
		// none of the current spokes accepted, create a new spoke for this job's bounds
		jobBound := j.AsBound(h.spokeSpan)
		logrus.WithFields(
			logrus.Fields{
				"start": jobBound.start,
				"end":   jobBound.end}).Debug("Creating new spoke to accomodate job")
		s := NewSpoke(jobBound.start, jobBound.end)
		j := s.AddJob(j)
		if j != nil {
			logrus.WithField("JobID", j.id).Panic("Hub should always accept a job. No spoke accepted")
		}
		h.AddSpoke(s)
	}
}

func (h *Hub) addToSpokes(j *Job) { h.addToSpokesFast(j) }

func (h *Hub) addToSpokesSlow(j *Job) {
	// Traverse in order
	acceped := false
	pq := &PriorityQueue{}
	heap.Init(pq)

	heap.Init(h.spokes)

	// Take the items out; they arrive in decreasing priority order.
	for h.spokes.Len() > 0 {
		i := heap.Pop(h.spokes).(*Item)
		// Add it to the new pq - this should be cheap because we are only arranging pointers
		heap.Push(pq, i)

		s := i.value.(*Spoke)
		j = s.AddJob(j)

		if j == nil {
			acceped = true
			break
		}
	}
	// Merge the items we extracted back into the main PQ
	for pq.Len() > 0 {
		heap.Push(h.spokes, heap.Pop(pq))
	}
	if !acceped {
		// none of the current spokes accepted, create a new spoke for this job's bounds
		jobBound := j.AsBound(h.spokeSpan)
		logrus.WithFields(
			logrus.Fields{
				"start": jobBound.start,
				"end":   jobBound.end}).Debug("Creating new spoke to accomodate job")
		s := NewSpoke(jobBound.start, jobBound.end)
		j := s.AddJob(j)
		if j != nil {
			logrus.WithField("JobID", j.id).Panic("Hub should always accept a job. No spoke accepted")
		}
		h.AddSpoke(s)
	}
}

// Status prints the state of the spokes of this hub
func (h *Hub) Status() {
	logrus.Info("-------------------------------------------------------------")
	logrus.Infof("Hub has %d spokes", len(h.spokeMap))
	logrus.Infof("Hub has %d total jobs", h.PendingJobsCount())
	logrus.Infof("Hub has %d removed jobs", h.removedJobsCount)
	logrus.Infof("Past spoke has %d jobs", h.pastSpoke.PendingJobsLen())
	for _, s := range h.spokeMap {
		logrus.Debugf("Spoke %s has %d jobs", s.id, s.PendingJobsLen())
		logrus.Debugf("Spoke %s start: %s end %s", s.id, s.start.String(), s.end.String())
	}
	logrus.Info("-------------------------------------------------------------")
}
