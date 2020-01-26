package monitor

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"
)

// MemAccountable struct is one that wishes to enable mem accounting for itself
type MemAccountable interface {
	SizeOf() uintptr
}

// MemMonitor watches memory usages and provides alarms when usage breaches thresholds
// Users of the monitor should call Fence() to stall operations while alarms are breached
// Fence blocks till accounted memory usage falls below the recovery threshold which is lower
// than the actual threshold to enable some breathing room
type MemMonitor interface {
	Increment(MemAccountable)
	Decrement(MemAccountable)
	Fence()
}

type noopMemMonitor struct{}

type memMonitor struct {
	current uintptr

	threshold         uintptr
	recoveryThreshold uintptr

	breachCond *sync.Cond
}

var memMonitorInstance MemMonitor

func init() {
	// configure mem manager
	if _, ok := os.LookupEnv("MEM_MAN_DISABLED"); ok {
		UseNoopMemMonitor()
		return
	}

	var threshold uint64
	if t, ok := os.LookupEnv("MEM_MAN_SIZE"); ok {
		var err error
		threshold, err = strconv.ParseUint(t, 10, 64)
		if err != nil {
			log.Info().Msgf("Unparseable mem alarm size specified: %s Should be specified as number of bytes", t)
		}
		if threshold == 0 {
			UseNoopMemMonitor()
			return
		}
		configureMemMonitor(uintptr(threshold))
		return
	}
	UseNoopMemMonitor()
}

// GetMemMonitor returns a ready mem monitor - Call ConfigureMemMonitor before this
// otherwise this method will panic
func GetMemMonitor() MemMonitor {
	if memMonitorInstance == nil {
		log.Fatal().Msg("GetMemMonitor called before ConfigureMemMonitor")
	}
	return memMonitorInstance
}

// configureMemMonitor sets a threshold value on a mem monitor.
// Call this before GetMemMonitor and before allocating the observed structs.
func configureMemMonitor(threshold uintptr) MemMonitor {
	if memMonitorInstance == nil {
		mm := &memMonitor{
			threshold:         threshold,
			recoveryThreshold: threshold - 4*(threshold>>10),
			breachCond:        sync.NewCond(&sync.Mutex{}),
		}
		log.Info().
			Str("AlarmThreshold", fmt.Sprintf("%d", mm.threshold)).
			Str("AlarmRecoveryThreshold", fmt.Sprintf("%d", mm.recoveryThreshold)).
			Msg("Initialized new memory monitor")
		memMonitorInstance = mm
	}
	return memMonitorInstance
}

func (mm *memMonitor) Increment(a MemAccountable) {
	atomic.AddUintptr(&mm.current, a.SizeOf())
	if atomic.LoadUintptr(&mm.current) >= mm.threshold {
		log.Warn().Msg("MemManager: breached mem threshold")
	}
}

func (mm *memMonitor) Decrement(a MemAccountable) {
	if atomic.AddUintptr(&mm.current, ^(a.SizeOf()-1)) < mm.recoveryThreshold {
		mm.breachCond.L.Lock()
		log.Warn().Msg("MemManager: recovered mem threshold")
		mm.breachCond.Broadcast()
		mm.breachCond.L.Unlock()
	}
}
func (mm *memMonitor) Fence() {
	// Fence blocks if above threshold
	mm.breachCond.L.Lock()
	for atomic.LoadUintptr(&mm.current) >= mm.threshold {
		mm.breachCond.Wait()
	}
	mm.breachCond.L.Unlock()
}

// UseNoopMemMonitor creates a noop implementation of mem-monitor if we want to fully disable it
// with minimal penalty
func UseNoopMemMonitor() {
	memMonitorInstance = &noopMemMonitor{}
	log.Info().Msg("Using NOOP memory monitor")
}

func (n *noopMemMonitor) Increment(a MemAccountable) {}
func (n *noopMemMonitor) Decrement(a MemAccountable) {}
func (n *noopMemMonitor) Fence()                     {}
