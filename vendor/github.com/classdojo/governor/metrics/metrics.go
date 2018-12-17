package metrics

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/DataDog/datadog-go/statsd"
)

type metricsWrapper struct {
	client *statsd.Client
	name   string
}

// Counter allows counting metric events
type Counter struct {
	metricsWrapper
}

// Timing metric provides a timer
type Timing struct {
	metricsWrapper
}

// TimingHistogram provides a timer that is consistent with how Datadog
// handles timers
type TimingHistogram struct {
	metricsWrapper
}

// Metrics instance is globally available once SetupMetrics is called
// and provides a statsd.Client
var Metrics *statsd.Client

// Hostname is a convenience variable that provides current hostname
var Hostname, _ = os.Hostname()

// SetupMetrics should be called once to ensure the metrics connection
// is setup
func SetupMetrics(isProd bool, namespace string) {
	SetupMetricsWithAddr(isProd, namespace, "127.0.0.1", 8125)
}

// SetupMetricsWithAddr creates a metrics sender with a custom remote address
func SetupMetricsWithAddr(isProd bool, namespace, host string, port int) {
	var err error
	if isProd {
		Metrics, err = statsd.New(fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			log.Fatal(err)
		}
		// prefix every metric with the app name
		if !strings.HasSuffix(namespace, ".") {
			namespace = fmt.Sprintf("%s.", namespace)
		}
		Metrics.Namespace = namespace
	} else {
		Metrics, err = statsd.NewWithWriter(&devWriter{})
	}
}

// NewCounter creates a Counter that can be Incremented or Decremented.
// Eg: c := NewCounter("prefix")
// c.Inc(1)
func NewCounter(prefix string) Counter {
	c := Counter{}
	c.name = prefix
	c.client = Metrics
	return c
}

// NewTiming creates a timer that can be started and stopped
func NewTiming(prefix string) Timing {
	t := Timing{}
	t.name = prefix
	t.client = Metrics
	return t
}

// Start begins recording time elapsed for the timer
// Calling Start again without calling stop will reset the timer
func (t *Timing) Start() time.Time {
	return time.Now()
}

// Stop ends recording elapsed time for the timer and sends the timing metric
func (t *Timing) Stop(startedAt time.Time) {
	val := time.Now().Sub(startedAt)
	t.client.Timing(t.name, val, nil, 1)
}

// NewTimingHistogram creates a timing histogram - use this for timing especially
// if you want to send timing metrics to datadog
func NewTimingHistogram(prefix string) TimingHistogram {
	th := TimingHistogram{}
	th.name = prefix
	th.client = Metrics
	return th
}

// Start beings recording elapsed time for the timer
// Calling Start again without calling stop will reset the timer
func (th *TimingHistogram) Start() time.Time {
	return time.Now()
}

// Measure ends recording elapsed time for the timer and sends the timing metric
// This function takes a start Time which can be provided either from TimingHistogram.Start()
// or any other source.
func (th *TimingHistogram) Measure(startedAt time.Time) {
	// milliseconds
	val := time.Now().Sub(startedAt).Seconds() * 1000.0
	th.client.Histogram(th.name, val, nil, 1)
}

// SimpleEvent sends a simple event to Datadog and appends the hostname
func SimpleEvent(title, text string) {
	textWithHost := fmt.Sprintf("%s Host:%s", text, Hostname)
	Metrics.SimpleEvent(title, textWithHost)
}

// Incr increments the counter by i
func (c *Counter) Incr(i int) {
	c.client.Incr(c.name, nil, 1)
}

// Decr decrements the counter by i
func (c *Counter) Decr(i int) {
	c.client.Decr(c.name, nil, 1)
}

// devWriter is a stats writer that will print metrics to stdout
// Useful for testing etc
type devWriter struct{}

func (d *devWriter) Write(data []byte) (n int, err error) {
	log.WithField("metric", string(data)).Info("Dev stats")
	return len(data), nil
}

func (d *devWriter) SetWriteTimeout(time.Duration) error {
	return nil
}

func (d *devWriter) Close() error {
	return nil
}
