/*
Package metrics provides a metrics wrapper.

Metrics must be initialized at startup as early as possible:
  import "github.com/chronomq/chronomq/pkg/metrics"
  ...
  metrics.InitMetrics(metricsCollectorAddr)
*/
package metrics
