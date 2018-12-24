package goyaad

import (
	"container/heap"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urjitbhatia/goyaad/pkg/metrics"
	"github.com/urjitbhatia/goyaad/pkg/persistence"
)

const (
	hundredYears = time.Hour * 24 * 365 * 100
)

// Hub is a time ordered collection of spokes
type Hub struct {
	spokeSpan time.Duration
	spokeMap  map[SpokeBound]*Spoke // quick lookup map
	spokes    *PriorityQueue

	pastSpoke    *Spoke // Permanently pinned to the past
	currentSpoke *Spoke // The current spoke

	reservedJobs map[string]*Job // This could also be a spoke that order by TTL - optimize later

	removedJobsCount uint64
	lock             *sync.Mutex

	persister persistence.Persister
}

// NewHub creates a new hub where adjacent spokes lie at the given
// spokeSpan duration boundary.
func NewHub(spokeSpan time.Duration, persister persistence.Persister) *Hub {
	h := &Hub{
		spokeSpan:        spokeSpan,
		spokeMap:         make(map[SpokeBound]*Spoke),
		spokes:           &PriorityQueue{},
		pastSpoke:        NewSpoke(time.Now().Add(-1*hundredYears), time.Now().Add(hundredYears)),
		currentSpoke:     nil,
		reservedJobs:     make(map[string]*Job),
		removedJobsCount: 0,
		lock:             &sync.Mutex{},
		persister:        persister,
	}
	heap.Init(h.spokes)

	logrus.WithFields(logrus.Fields{
		"start": h.pastSpoke.start.String(),
		"end":   h.pastSpoke.end.String(),
	}).Debug("Created hub with past spoke")

	go h.StatusPrinter()

	return h
}

// PendingJobsCount return the number of jobs currently pending
func (h *Hub) PendingJobsCount() int {
	count := h.pastSpoke.PendingJobsLen()
	for _, v := range h.spokeMap {
		count += v.PendingJobsLen()
	}

	return count
}

// ReservedJobsCount return the number of jobs currently reserved
func (h *Hub) ReservedJobsCount() int {
	return len(h.reservedJobs)
}

// CancelJob cancels a job if found. Calls are noop for unknown jobs
func (h *Hub) CancelJob(jobID string) error {
	metrics.Incr("hub.cancel.req")
	h.lock.Lock()
	defer h.lock.Unlock()

	logrus.Debug("cancel: ", jobID)
	// Search if this job is reserved
	if _, ok := h.reservedJobs[jobID]; ok {
		logrus.Debug("found in reserved: ", jobID)
		delete(h.reservedJobs, jobID)
		h.removedJobsCount++
		return nil
	}

	s, err := h.FindOwnerSpoke(jobID)
	if err != nil {
		logrus.Debug("cancel found no owner spoke: ", jobID)
		return err
	}
	logrus.Debug("cancel found owner spoke: ", jobID)
	s.CancelJob(jobID)
	h.removedJobsCount++
	metrics.Incr("hub.cancel.ok")
	return nil
}

// FindOwnerSpoke returns the spoke that owns this job
func (h *Hub) FindOwnerSpoke(jobID string) (*Spoke, error) {

	if h.pastSpoke.OwnsJob(jobID) {
		return h.pastSpoke, nil
	}

	// Checking the current spoke - lock the hub
	if h.currentSpoke != nil && h.currentSpoke.OwnsJob(jobID) {
		return h.currentSpoke, nil
	}

	for _, v := range h.spokeMap {
		if v.OwnsJob(jobID) {
			return v, nil
		}
	}
	return nil, errors.New("Cannot find job owner spoke")
}

// addSpoke adds spoke s to this hub
func (h *Hub) addSpoke(s *Spoke) {
	h.spokeMap[s.SpokeBound] = s
	heap.Push(h.spokes, s.AsPriorityItem())
}

// Next returns the next job that is ready now or returns nil.
func (h *Hub) Next() *Job {
	defer metrics.Time("hub.next.search.duration", time.Now())

	h.lock.Lock()
	defer h.lock.Unlock()

	pastLocker := h.pastSpoke.GetLocker()
	pastLocker.Lock()
	defer pastLocker.Unlock()

	// Find a job in past spoke
	j := h.pastSpoke.Next()
	if j != nil {
		h.reserve(j)
		logrus.Debug("Got job from past spoke")
		return j
	}
	// Checked past spoke

	// Find a job in current spoke
	// If current is empty and now expired, prune it...
	if h.currentSpoke != nil {
		if h.currentSpoke.PendingJobsLen() == 0 && h.currentSpoke.AsTemporalState() == Past {
			logrus.Info("pruning the current spoke")
			// This routine could be unfortunate - it found a currentspoke that was expired
			// so it has the pay the price finding the next candidate
			delete(h.spokeMap, h.currentSpoke.SpokeBound)
			h.currentSpoke = nil
		}
	}

	// No currently assigned spoke
	if h.currentSpoke == nil {
		// Fix the heap
		heap.Init(h.spokes)

		if h.spokes.Len() == 0 {
			// No spokes - can't do anything. Return
			return nil
		}

		// New current candidate
		item := h.spokes.AtIdx(0)
		current := item.value.(*Spoke)
		switch current.AsTemporalState() {
		case Future:
			// Next in time is still not current. Can't do anything. Return
			return nil
		case Past, Current:
			// We have found a new current spoke
			h.currentSpoke = current
			// Pop it from the queue - this is now a current spoke
			heap.Pop(h.spokes)
			logrus.Infof("Hub spoke pop cap: %d", h.spokes.Cap())
		}
	}

	// Read from current spoke

	// Assert - At this point, hub should have a current spoke
	if h.currentSpoke == nil {
		logrus.Panic("Unreachable state :: hub has a nil spoke after candidate search")
	}

	currentLocker := h.currentSpoke.GetLocker()
	currentLocker.Lock()
	defer currentLocker.Unlock()

	j = h.currentSpoke.Next()
	if j == nil {
		// no job - return
		logrus.Debug("No job in current spoke")
		return nil
	}

	logrus.Debug("reserving job: ", j.id)
	h.reserve(j)

	return j
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
func (h *Hub) AddJob(j *Job) error {
	defer metrics.Time("hub.job.add.duration", time.Now())
	metrics.GaugeInt("hub.job.size", len(j.body))

	switch j.AsTemporalState() {
	case Past:
		logrus.Tracef("Adding job: %s to past spoke", j.id)
		pastLocker := h.pastSpoke.GetLocker()
		pastLocker.Lock()
		defer pastLocker.Unlock()

		logrus.WithField("JobID", j.ID).Trace("Adding job to past spoke")
		err := h.pastSpoke.AddJob(j)
		if err != nil {
			logrus.WithError(err).Error("Past spoke rejected job. This should never happen")
			return err
		}
		metrics.Incr("hub.addjob.past")
	case Future:
		logrus.Tracef("Adding job: %s to future spoke", j.id)
		// Lock hub so that current spoke isn't replaced
		h.lock.Lock()
		defer h.lock.Unlock()

		// Lock current spoke so that add fixes the PQ as it adds
		if h.currentSpoke != nil {
			currLocker := h.currentSpoke.GetLocker()
			currLocker.Lock()
			defer currLocker.Unlock()

			if h.currentSpoke.ContainsJob(j) {
				err := h.currentSpoke.AddJob(j)
				if err != nil {
					logrus.WithError(err).Error("Current spoke rejected job. This should never happen")
					return err
				}
				return nil
			}
		}

		// Search for a spoke that can take ownership of this job
		// Reads are still going to be ordered anyways
		jobBound := j.AsBound(h.spokeSpan)
		candidate, ok := h.spokeMap[jobBound]
		if ok {
			// Found a candidate that can take this job
			logrus.Debugf("Adding job: %s to candidate spoke", j.id)
			err := candidate.AddJob(j)
			if err != nil {
				logrus.WithError(err).Error("Hub should always accept a job. No spoke accepted")
				return err
			}
			// Accepted, all done...
			return nil
		}

		// Time to create a new spoke for this job
		logrus.Debugf("Adding job: %s to a new spoke", j.id)
		s := NewSpoke(jobBound.start, jobBound.end)
		err := s.AddJob(j)
		if err != nil {
			logrus.WithError(err).Error("Hub should always accept a job. No spoke accepted")
			return err
		}

		// h is still locked here so it's ok
		h.addSpoke(s)
	}
	metrics.Incr("hub.addjob")
	return nil
}

// Status prints the state of the spokes of this hub
func (h *Hub) Status() {
	logrus.Info("-------------------------------------------------------------")

	spokesCount := len(h.spokeMap)
	logrus.Infof("Hub has %d spokes", spokesCount)
	metrics.GaugeInt("hub.spoke.count", spokesCount)

	h.lock.Lock()
	defer h.lock.Unlock()

	pendingJobCount := h.PendingJobsCount()
	logrus.Infof("Hub has %d total jobs", pendingJobCount)
	metrics.GaugeInt("hub.job.count", pendingJobCount)

	reservedJobCount := len(h.reservedJobs)
	logrus.Infof("Hub has %d reserved jobs", reservedJobCount)
	metrics.GaugeInt("hub.job.reserved.count", reservedJobCount)

	logrus.Infof("Hub has %d removed jobs", h.removedJobsCount)
	metrics.Gauge("hub.job.removed.count", float64(h.removedJobsCount))

	logrus.Infof("Past spoke has %d jobs", h.pastSpoke.PendingJobsLen())
	metrics.GaugeInt("hub.job.pastspoke.count", h.pastSpoke.PendingJobsLen())

	if h.currentSpoke != nil {
		logrus.Infof("Current spoke has %d jobs", h.currentSpoke.PendingJobsLen())
		metrics.GaugeInt("hub.job.pastspoke.count", h.currentSpoke.PendingJobsLen())
	}

	logrus.Infof("Assigned current spoke: %v", h.currentSpoke == nil)
	logrus.Info("-------------------------------------------------------------")
}

// StatusPrinter starts a status printer that prints hub stats over some time interval
func (h *Hub) StatusPrinter() {
	t := time.NewTicker(time.Minute * 2)
	for range t.C {
		h.Status()
	}
}

// Persist locks the hub and starts persisting data to disk
func (h *Hub) Persist() chan error {
	h.lock.Lock()
	defer h.lock.Unlock()

	ec := make(chan error)

	persistSpoke := func(s *Spoke) int {
		j := 0
		for j = 0; j < s.jobQueue.Len(); j++ {
			job := s.jobQueue.AtIdx(j).value.(*Job)

			err := job.Persist(h.persister)
			if err != nil {
				logrus.Error(errors.Wrap(err, "Persister failed to save job"))
				ec <- err
				continue
			}
		}

		return j
	}

	go func() {
		defer close(ec)

		logrus.Warn("Starting disk offload")
		logrus.Warnf("Total spokes: %d Total jobs: %d", h.spokes.Len(), h.PendingJobsCount())
		var i int

		wg := &sync.WaitGroup{}
		wg.Add(h.spokes.Len())
		for i = 0; i < h.spokes.Len(); i++ {
			s := h.spokes.AtIdx(i).value.(*Spoke)
			func() {
				defer wg.Done()
				j := persistSpoke(s)
				logrus.Infof("Persisted %d jobs from spoke %d", j, i)
			}()
		}

		// Save past spoke
		p := persistSpoke(h.pastSpoke)
		logrus.Infof("Persisted %d jobs from past spoke", p)

		// Save current spoke
		if h.currentSpoke != nil {
			c := persistSpoke(h.currentSpoke)
			logrus.Infof("Persisted %d jobs from current spoke", c)
		}

		// Save the reserved jobs
		for _, j := range h.reservedJobs {
			if err := j.Persist(h.persister); err != nil {
				logrus.Error(errors.Wrap(err, "Persister failed to save reserved job"))
				ec <- err
				continue
			}
		}

		wg.Wait()
		h.persister.Finalize()
	}()

	return ec
}
