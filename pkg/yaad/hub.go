package yaad

import (
	"container/heap"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/seiflotfy/cuckoofilter"

	"github.com/urjitbhatia/yaad/pkg/yaad/stats"
	"github.com/urjitbhatia/yaad/pkg/metrics"
	"github.com/urjitbhatia/yaad/pkg/persistence"
)

const (
	hundredYears = time.Hour * 24 * 365 * 100

	// DefaultMaxCFSize is the max number of entries the filter is expected to hold
	// creates a Cuckoo filter taking 512MB fixed space overhead on a 64 bit machine
	// with an average job size of 1Kb, that is 500*1000_000_000/1024/1024/1024=465.66128730773926 GB of just job payload storage
	// so this upper-bound should be enough
	DefaultMaxCFSize uint = 500 * 1000 * 1000

	// TestMaxCFSize value should be used for testing
	TestMaxCFSize uint = 10000
)

// HubOpts define customizations for Hub initialization
type HubOpts struct {
	Persister      persistence.Persister // persister to store/restore from disk
	AttemptRestore bool                  // If true, hub will try to restore from disk on start
	SpokeSpan      time.Duration         // How wide should the spokes be
	MaxCFSize      uint                  // Max size of the Cuckoo Filter
}

// Hub is a time ordered collection of spokes
type Hub struct {
	jobFilter *cuckoo.Filter
	spokeSpan time.Duration         // How much time does a spoke span
	spokeMap  map[SpokeBound]*Spoke // Quick lookup map
	spokes    *PriorityQueue        // Actual spokes sorted by time

	pastSpoke    *Spoke // Permanently pinned to the past
	currentSpoke *Spoke // The current spoke - started in the past or now, ends in the future or now

	stats *stats.Counters
	lock  *sync.Mutex

	persister persistence.Persister
}

// NewHub creates a new hub where adjacent spokes lie at the given
// spokeSpan duration boundary.
func NewHub(opts *HubOpts) *Hub {
	maxCFSize := TestMaxCFSize // keep test mem requirements low by default
	if opts.MaxCFSize != 0 {
		maxCFSize = opts.MaxCFSize
	}
	h := &Hub{
		jobFilter:    cuckoo.NewFilter(maxCFSize),
		spokeSpan:    opts.SpokeSpan,
		spokeMap:     make(map[SpokeBound]*Spoke),
		spokes:       &PriorityQueue{},
		pastSpoke:    NewSpoke(time.Now().Add(-1*hundredYears), time.Now().Add(hundredYears)),
		currentSpoke: nil,
		stats:        &stats.Counters{},
		lock:         &sync.Mutex{},
		persister:    opts.Persister,
	}
	heap.Init(h.spokes)

	log.Info().Dur("spokeSpan", opts.SpokeSpan).
		Bool("attemptRestore", opts.AttemptRestore).
		Uint("maxCFSize", maxCFSize).
		Msg("Created hub")

	go func() {
		if opts.AttemptRestore {
			log.Info().Msg("Hub: Entering restore mode")
			err := h.Restore()
			if err != nil {
				log.Error().Err(err).Msg("Hub: Restore error")
			}

			log.Info().Msg("Hub: Initial restore finished. Resuming")
		}
	}()
	go h.StatusPrinter()

	return h
}

// Stop the hub gracefully and if persist is true, then persist all jobs to disk for later recovery
func (h *Hub) Stop(persist bool) {
	if persist {
		log.Info().Int("PID", os.Getpid()).Msg("Hub:Stop Starting persistence")
		errC := h.PersistLocked()
		errCount := 0
		for range errC {
			errCount++
		}
		log.Info().Int("errorCount", errCount).Msg("Hub:Stop Finished persistence with errors")
	}
	log.Info().Msg("Hub:Stop stopped")
}

// Stats returns a snapshot of the current hubs stats
func (h *Hub) Stats() stats.Snapshot {
	return h.stats.Read()
}

// CancelJobLocked cancels a job if found. Calls are noop for unknown jobs
func (h *Hub) CancelJobLocked(jobID string) error {
	go metrics.Incr("hub.cancel.req")
	h.lock.Lock()
	defer h.lock.Unlock()
	id := []byte(jobID)
	if !h.jobFilter.Lookup(id) {
		// no such job
		go metrics.Incr("hub.cancel.ok")
		return nil
	}
	err := h.cancelJob(jobID)
	if err != nil {
		h.jobFilter.Delete(id)
	}
	return err
}

func (h *Hub) cancelJob(jobID string) error {
	log.Debug().Str("jobID", jobID).Msg("canceling job")

	s, err := h.findOwnerSpoke(jobID)
	if err != nil {
		log.Debug().Str("jobID", jobID).Msg("cancel found no owner spoke")
		// return nil - cancel if job not found is idempotent
		return nil
	}
	log.Debug().Str("jobID", jobID).Msg("cancel found owner spoke")
	err = s.CancelJobLocked(jobID)
	if err == nil {
		h.stats.DecrJob()
		go metrics.Incr("hub.cancel.ok")
	}
	return err
}

// findOwnerSpoke returns the spoke that owns this job. Lock the hub before calling this
func (h *Hub) findOwnerSpoke(jobID string) (*Spoke, error) {
	if h.pastSpoke.OwnsJobLocked(jobID) {
		return h.pastSpoke, nil
	}

	// Checking the current spoke
	if h.currentSpoke != nil && h.currentSpoke.OwnsJobLocked(jobID) {
		return h.currentSpoke, nil
	}

	// Find the owner in the spoke map
	for _, s := range h.spokeMap {
		if s.OwnsJobLocked(jobID) {
			return s, nil
		}
	}
	return nil, errors.New("Cannot find job owner spoke")
}

// addSpoke adds spoke s to this hub
func (h *Hub) addSpoke(s *Spoke) {
	defer h.stats.IncrSpoke()
	h.spokeMap[s.SpokeBound] = s
	heap.Push(h.spokes, s.AsPriorityItem())
}

// deleteSpokeFromMap removes a spoke from the map
func (h *Hub) deleteSpokeFromMap(s *Spoke) {
	defer h.stats.DecrSpoke()
	delete(h.spokeMap, s.SpokeBound)
}

// NextLocked returns the next job that is ready now or returns nil.
func (h *Hub) NextLocked() *Job {
	defer metrics.Time("hub.next.search.duration", time.Now())

	h.lock.Lock()
	defer h.lock.Unlock()
	j := h.next()
	if j != nil {
		h.jobFilter.Delete([]byte(j.id))
	}

	return j
}
func (h *Hub) next() *Job {

	// since we have the lock, send some metrics
	go metrics.GaugeInt("hub.job.count", int(h.stats.Read().CurrentJobs))
	go metrics.GaugeInt("hub.spoke.count", h.spokes.Len())

	// Lock Past spoke lock in func scope
	if j := func() *Job {
		go metrics.GaugeInt("hub.job.pastspoke.count", h.pastSpoke.PendingJobsLen())

		// Find a job in past spoke
		j := h.pastSpoke.NextLocked()
		if j != nil {
			log.Debug().Msg("Got job from past spoke")
		}
		return j
	}(); j != nil {
		h.stats.DecrJob()
		return j
	}

	// Checked past spoke

	// Find a job in current spoke
	// If current is empty and now expired, prune it...
	if h.currentSpoke != nil {
		h.currentSpoke = func() *Spoke {
			if h.currentSpoke.PendingJobsLen() == 0 && h.currentSpoke.AsTemporalState() == Past {
				// This routine could be unfortunate - it found a currentspoke that was expired
				// so it has the pay the price finding the next candidate
				h.deleteSpokeFromMap(h.currentSpoke)
				return nil
			}
			return h.currentSpoke
		}()
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
		}
	}

	// Read from current spoke

	// Assert - At this point, hub should have a current spoke
	if h.currentSpoke == nil {
		log.Panic().Msg("Unreachable state :: hub has a nil spoke after candidate search")
	}

	go metrics.GaugeInt("hub.job.currentspoke.count", h.currentSpoke.PendingJobsLen())

	j := h.currentSpoke.NextLocked()
	if j == nil {
		// no job - return
		log.Debug().Msg("No job in current spoke")
		return nil
	}

	log.Debug().Str("jobID", j.id).Msg("returning next job")
	h.stats.DecrJob()
	return j
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
	for _, s := range h.spokeMap {
		if s.IsExpired() && s.PendingJobsLen() == 0 {
			h.deleteSpokeFromMap(s)
		}
		pruned++
	}
	return pruned
}

// AddJobLocked to this hub. Hub should never reject a job - this method will panic if that happens
func (h *Hub) AddJobLocked(j *Job) error {
	defer metrics.Time("hub.job.add.duration", time.Now())
	go metrics.GaugeInt("hub.job.size", len(j.body))

	h.lock.Lock()
	defer h.lock.Unlock()

	// Check is job already exists in the system
	id := []byte(j.id)
	if h.jobFilter.Lookup(id) {
		// filter can give us false positives, do a full scan
		if spoke, _ := h.findOwnerSpoke(j.id); spoke != nil {
			return fmt.Errorf("Rejecting job. Job with ID: %s already exists", j.id)
		}
	}

	err := h.addJob(j)
	if err == nil {
		if !h.jobFilter.Insert(id) {
			log.Error().Msgf("Could not insert into the filter. ID: %s", id)
		}
		h.stats.IncrJob()
		go metrics.Incr("hub.addjob")
	}
	return err
}

func (h *Hub) addJob(j *Job) error {
	switch j.AsTemporalState() {
	case Past:
		log.Debug().Str("jobID", j.id).Msg("Adding job to past spoke")
		err := h.pastSpoke.AddJobLocked(j)
		if err != nil {
			log.Error().Err(err).Msg("Past spoke rejected job. This should never happen")
			return err
		}
		go metrics.Incr("hub.addjob.past")
		return nil
	case Future:
		log.Debug().Str("jobID", j.id).Msg("Adding job to future spoke")
		// Lock current spoke so that add fixes the PQ as it adds
		if h.currentSpoke != nil {
			if h.currentSpoke.ContainsJob(j) {
				err := h.currentSpoke.AddJobLocked(j)
				if err != nil {
					log.Error().Err(err).Msg("Current spoke rejected job. This should never happen")
					return err
				}
				return nil
			}
		}

		// Search for a spoke that can take ownership of this job
		// Reads are still going to be ordered anyways
		jobBound := j.AsBound(h.spokeSpan)
		if candidateSpoke, ok := h.spokeMap[jobBound]; ok {
			// Found a candidate that can take this job
			log.Debug().Str("jobID", j.id).Msg("Adding job to candidate spoke")
			err := candidateSpoke.AddJobLocked(j)
			if err != nil {
				log.Error().Err(err).Msg("Hub should always accept a job. No spoke accepted ")
				return err
			}
			// Accepted, all done...
			return nil
		}

		// Time to create a new spoke for this job
		log.Debug().Str("jobID", j.id).Msg("Adding job to a new spoke")
		s := NewSpoke(jobBound.start, jobBound.end)
		err := s.AddJobLocked(j)
		if err != nil {
			log.Error().Err(err).Msg("Hub should always accept a job. No spoke accepted ")
			return err
		}
		// h is still locked here so it's ok
		h.addSpoke(s)
		return nil
	}

	err := errors.Errorf("Unable to find a spoke for job. This should never happen")
	log.Error().Err(err).Msg("Cant add job to hub")
	return err
}

// StatusLocked prints the state of the spokes of this hub
func (h *Hub) StatusLocked() {
	log.Info().Msg("------------------------Hub Stats----------------------------")

	hubStats := h.stats.Read()
	log.Info().Int64("spokesCount", hubStats.CurrentSpokes).Send()
	go metrics.GaugeInt("hub.spoke.count", int(hubStats.CurrentSpokes))

	log.Info().Int64("pendingJobsCount", hubStats.CurrentJobs).Send()
	go metrics.GaugeInt("hub.job.count", int(hubStats.CurrentJobs))

	log.Info().Int64("removedJobsCount", hubStats.RemovedJobs).Send()
	go metrics.GaugeInt("hub.job.removed.count", int(hubStats.RemovedJobs))

	// lock only for this bit - current spoke can be replaced while running...
	h.lock.Lock()
	defer h.lock.Unlock()
	log.Info().Int("pastSpokePendingJobsCount", h.pastSpoke.PendingJobsLen()).Send()
	log.Info().Uint("jobFilterCount", h.jobFilter.Count()).Send()
	go metrics.GaugeInt("hub.job.pastspoke.count", h.pastSpoke.PendingJobsLen())

	if h.currentSpoke != nil {
		log.Info().Int("currentSpokePendingJobsCount", h.currentSpoke.PendingJobsLen()).Send()
		go metrics.GaugeInt("hub.job.currentspoke.count", h.currentSpoke.PendingJobsLen())
	}
	log.Info().Msg("-------------------------------------------------------------")
}

// StatusPrinter starts a status printer that prints hub stats over some time interval
func (h *Hub) StatusPrinter() {
	t := time.NewTicker(time.Second * 10)
	for range t.C {
		h.StatusLocked()
	}
}

// PersistLocked locks the hub and starts persisting data to disk
func (h *Hub) PersistLocked() chan error {
	log.Warn().Msg("Starting disk offload")
	wg := &sync.WaitGroup{}
	ec := make(chan error)

	h.lock.Lock()
	go func() {
		defer h.lock.Unlock()

		log.Warn().
			Int("totalSpokes", h.spokes.Len()).
			Int64("pendingJobsCount", h.stats.Read().CurrentJobs).
			Msg("About to persist")
		wg.Add(h.spokes.Len())
		defer close(ec)

		for i := 0; i < h.spokes.Len(); i++ {
			s := h.spokes.AtIdx(i).value.(*Spoke)
			func() {
				defer wg.Done()
				errC := s.PersistLocked(h.persister)
				for e := range errC {
					ec <- e
				}
			}()
		}

		// Save past spoke
		errC := h.pastSpoke.PersistLocked(h.persister)
		for e := range errC {
			ec <- e
		}

		// Save current spoke
		if h.currentSpoke != nil {
			errC := h.currentSpoke.PersistLocked(h.persister)
			for e := range errC {
				ec <- e
			}
		}

		wg.Wait()
		h.persister.Finalize()
	}()

	return ec
}

// Restore loads any jobs saved to disk at the given path
func (h *Hub) Restore() error {
	jobs, err := h.persister.Recover()
	if err != nil {
		return err
	}

	errDecodeCount := 0
	errAddCount := 0
	recoverCount := 0
	for e := range jobs {
		j := new(Job)
		err := j.GobDecode(e)
		if err != nil {
			errDecodeCount++
			log.Error().Err(err).Send()
			continue
		}
		if err = h.AddJobLocked(j); err != nil {
			errAddCount++
			log.Error().Err(err).Send()
			continue
		}
		recoverCount++
	}
	log.Info().Int("recoverCount", recoverCount).Msg("Hub:Restore recovered entries")

	if errAddCount == 0 && errDecodeCount == 0 {
		return nil
	}

	var retErr = errors.New("Hub:Restore failed")
	retErr = errors.Wrapf(retErr, "Hub:Restore encountered %d errors decoding persisted jobs", errDecodeCount)
	retErr = errors.Wrapf(retErr, "Hub:Restore encountered %d errors adding persisted jobs", errAddCount)
	return retErr
}

// GetNJobs returns upto N jobs (or less if there are less jobs in available)
// It does not return a consistent snapshot of jobs but provides a best effort view
func (h *Hub) GetNJobs(n int) chan *Job {
	jobChan := make(chan *Job)
	go func() {
		defer close(jobChan)
		func() {
			// Iterate over jobs from the past spoke first
			h.pastSpoke.Lock()
			defer h.pastSpoke.Unlock()

			pastJobsLen := h.pastSpoke.jobQueue.Len()
			for i := 0; i < pastJobsLen; i++ {
				j := h.pastSpoke.jobQueue.AtIdx(i)
				jobChan <- j.Value().(*Job)
				n--
				if n <= 0 {
					return
				}
			}
		}()
		if n <= 0 {
			// Found all requested jobs
			return
		}

		// Iterate over the future spokes from the map (We dont care about the order in this case)
		for _, s := range h.spokeMap {
			s.Lock()
			defer s.Unlock()
			// Iterate over the jobs in this spoke
			jobsLen := s.jobQueue.Len()
			for i := 0; i < jobsLen; i++ {
				j := s.jobQueue.AtIdx(i)
				jobChan <- j.Value().(*Job)
				n--
				if n <= 0 {
					break
				}
			}
			// Continue if n is not 0 yet
		}
	}()

	return jobChan
}
