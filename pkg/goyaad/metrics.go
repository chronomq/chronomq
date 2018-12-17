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
		statsConn, err := statsd.New(statsAddr) // Connect to the UDP port 8125 by default.
		if err != nil {
			logrus.Fatal(err)
		}
		statsConn.Namespace = "goyaad"
		MetricsClient = statsConn
	}
}

// ReportTime sends timing metrics when called.
// Ideally used with defer as ReportTime("metricname", time.Now())
func ReportTime(name string, start time.Time) {
	MetricsClient.Timing(name, time.Now().Sub(start), nil, 0)
}
