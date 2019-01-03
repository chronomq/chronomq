package metrics

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/sirupsen/logrus"
)

// Client is the global metrics client for sending statsd compatible metrics
var Client *statsd.Client

// InitMetrics idempotently initializes a Client
func InitMetrics(statsAddr string) {
	if Client == nil {
		var err error
		Client, err = statsd.NewBuffered(statsAddr, 500)
		if err != nil {
			logrus.Fatal(err)
		}
		Client.Namespace = "goyaad"
	}
}

// Time sends timing metrics when called.
// Ideally used with defer as Time("metricname", time.Now())
func Time(name string, start time.Time) error {
	return Client.Timing(name, time.Now().Sub(start), nil, 1)
}

// Incr increments the given metric name
func Incr(name string) error {
	return Client.Incr(name, nil, 1)
}

// Decr decrements the given metric name
func Decr(name string) error {
	return Client.Decr(name, nil, 1)
}

// Gauge sends a gauge metric
func Gauge(name string, val float64) error {
	return Client.Gauge(name, val, nil, 1)
}

// GaugeInt sends a gauge metric
func GaugeInt(name string, val int) error {
	return Client.Gauge(name, float64(val), nil, 1)
}
