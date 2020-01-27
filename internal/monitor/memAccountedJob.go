package monitor

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"code.cloudfoundry.org/bytefmt"
	"github.com/rs/zerolog/log"
)

// MemAccountable struct is one that wishes to enable mem accounting for itself
type MemAccountable interface {
	SizeOf() uint64
}

// MemMonitor watches memory usages and provides alarms when usage breaches watermark
// Users of the monitor should call Fence() to stall operations while alarms are breached
// Fence blocks till accounted memory usage falls below the recovery watermark which is lower
// than the actual threshold to enable some breathing room
// It is safe to call methods on MemMonitor from multiple goroutines
type MemMonitor interface {
	// Increment adds the mem used by the given MemAccountable to the internal counter
	// Call this when initializing a new instance of that MemAccountable
	Increment(MemAccountable)
	// Decrement subtracts the mem used by the given MemAccountable from the internal counter
	// Call this when the last ref to that MemAccountable is given up
	Decrement(MemAccountable)
	// Fence blocks while mem usage accounted by MemManager is above the watermark
	// Multiple goroutines can call and be blocked by Fence
	Fence()
	// Breached returns true if mem usage is currently above the watermark and hasn't gone below recoveryWatermark yet
	Breached() bool
}

type memMonitor struct {
	current uint64 // currently accounted for memory usage

	watermark         uint64 // alarm watermark
	recoveryWatermark uint64 // alarm recovery watermark

	breachCond *sync.Cond
}

var memMonitorInstance MemMonitor

func init() {
	// configure mem manager
	t, ok := os.LookupEnv("MEM_HIGH_WATERMARK")
	if !ok {
		UseNoopMemMonitor()
		return
	}

	// If only number, treat as bytes
	watermark, err := strconv.ParseUint(t, 10, 64)
	if err == nil {
		configureMemMonitor(watermark)
		return
	}

	// try to parse with units
	watermark, err = bytefmt.ToBytes(t)
	if err != nil {
		log.Fatal().Msgf("Unparseable mem alarm size specified: %s Should be specified as number of bytes", t)
	}
	configureMemMonitor(watermark)
}

// GetMemMonitor returns a ready mem monitor - Call ConfigureMemMonitor before this
// otherwise this method will panic
func GetMemMonitor() MemMonitor {
	if memMonitorInstance == nil {
		log.Fatal().Msg("GetMemMonitor called before ConfigureMemMonitor")
	}
	return memMonitorInstance
}

// configureMemMonitor sets a watermark value on a mem monitor.
// Call this before GetMemMonitor and before allocating the observed structs.
func configureMemMonitor(watermark uint64) MemMonitor {
	if watermark == 0 {
		UseNoopMemMonitor()
		return memMonitorInstance
	}

	if memMonitorInstance == nil {
		mm := &memMonitor{
			watermark:         watermark,
			recoveryWatermark: watermark - 10*(watermark>>10),
			breachCond:        sync.NewCond(&sync.Mutex{}),
		}
		log.Info().
			Str("AlarmWatermark", bytefmt.ByteSize(mm.watermark)).
			Str("AlarmRecoveryWatermark", bytefmt.ByteSize(mm.recoveryWatermark)).
			Msg("Initialized new memory monitor")
		memMonitorInstance = mm
	}
	return memMonitorInstance
}

func (mm *memMonitor) Increment(a MemAccountable) {
	atomic.AddUint64(&mm.current, a.SizeOf())
}

func (mm *memMonitor) Decrement(a MemAccountable) {
	if atomic.AddUint64(&mm.current, ^(a.SizeOf()-1)) < mm.recoveryWatermark {
		mm.breachCond.L.Lock()
		mm.breachCond.Broadcast()
		mm.breachCond.L.Unlock()
	}
}

func (mm *memMonitor) Fence() {
	// Fence blocks if above watermark
	mm.breachCond.L.Lock()
	for atomic.LoadUint64(&mm.current) >= mm.watermark {
		mm.breachCond.Wait()
	}
	mm.breachCond.L.Unlock()
}

func (mm *memMonitor) Breached() bool {
	return atomic.LoadUint64(&mm.current) >= mm.watermark
}

// ############ NOOP Mem Monitor ################
type noopMemMonitor struct{}

// UseNoopMemMonitor creates a noop implementation of mem-monitor if we want to fully disable it
// with minimal penalty
func UseNoopMemMonitor() {
	memMonitorInstance = &noopMemMonitor{}
	log.Info().Msg("Using NOOP memory monitor")
}

func (n *noopMemMonitor) Increment(a MemAccountable) {}
func (n *noopMemMonitor) Decrement(a MemAccountable) {}
func (n *noopMemMonitor) Fence()                     {}
func (n *noopMemMonitor) Breached() bool             { return false }
