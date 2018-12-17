package goyaad

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/sirupsen/logrus"
)

// MetricsClient is the global metrics client for sending statsd compatible metrics
var MetricsClient *statsd.Client

// InitMetrics idempotently initializes a MetricsClient
func InitMetrics(statsAddr string) {
	if MetricsClient == nil {
		var err error
		MetricsClient, err = statsd.New(statsAddr)
		if err != nil {
			logrus.Fatal(err)
		}
		MetricsClient.Namespace = "goyaad"
	}
}

// ReportTime sends timing metrics when called.
// Ideally used with defer as ReportTime("metricname", time.Now())
func ReportTime(name string, start time.Time) {
	err := MetricsClient.Timing(name, time.Now().Sub(start), nil, 1)
	if err != nil {
		logrus.Error(err)
	}
}

// IncrMetric increments the given metric name
func IncrMetric(name string) error {
	return MetricsClient.Incr(name, nil, 1)
}

// DecrMetric decrements the given metric name
func DecrMetric(name string) error {
	return MetricsClient.Decr(name, nil, 1)
}
