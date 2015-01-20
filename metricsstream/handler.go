package metricsstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestlog"
)

type MetricsStream struct {
	executor *cuirass.CommandExecutor
}

func NewMetricsStream(e *cuirass.CommandExecutor) *MetricsStream {
	return &MetricsStream{
		executor: e,
	}
}

func (h *MetricsStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")

	for {
		metrics := h.executor.Metrics().All()
		if len(metrics) == 0 {
			fmt.Fprintln(w, "ping: ")
		} else {
			encoder := json.NewEncoder(w)
			for _, m := range metrics {
				w.Write([]byte("data: "))
				h.writeMetrics(m, encoder)
				w.Write([]byte("\n"))
			}
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		} else {
			panic("Flush not supported")
		}
		time.Sleep(2000 * time.Millisecond)
	}
}

func (h *MetricsStream) writeMetrics(m *metrics.CommandMetrics, e *json.Encoder) {
	metrics := struct {
		Type                                                     string         `json:"type"`
		Name                                                     string         `json:"name"`
		Group                                                    string         `json:"group"`
		CurrentTime                                              int            `json:"currentTime"`
		IsCircuitBreakerOpen                                     bool           `json:"isCircuitBreakerOpen"`
		ErrorPercentage                                          int            `json:"errorPercentage"`
		ErrorCount                                               int            `json:"errorCount"`
		RequestCount                                             int            `json:"requestCount"`
		RollingCountCollapsedRequests                            int            `json:"rollingCountCollapsedRequests"`
		RollingCountExceptionsThrown                             int            `json:"rollingCountExceptionsThrown"`
		RollingCountFailure                                      int            `json:"rollingCountFailure"`
		RollingCountFallbackFailure                              int            `json:"rollingCountFallbackFailure"`
		RollingCountFallbackRejection                            int            `json:"rollingCountFallbackRejection"`
		RollingCountFallbackSuccess                              int            `json:"rollingCountFallbackSuccess"`
		RollingCountResponseFromCache                            int            `json:"rollingCountResponsesFromCache"`
		RollingCountSemaphoreRejected                            int            `json:"rollingCountSemaphoreRejected"`
		RollingCountShortCircuited                               int            `json:"rollingCountShortCircuited"`
		RollingCountSuccess                                      int            `json:"rollingCountSuccess"`
		RollingCountThreadPoolRejected                           int            `json:"rollingCountThreadPoolRejected"`
		RollingCountTimeout                                      int            `json:"rollingCountTimeout"`
		CurrentConcurrentExecutionCount                          int            `json:"currentConcurrentExecutionCount"`
		LatencyExecuteMean                                       int            `json:"latencyExecute_mean"`
		LatencyExecute                                           map[string]int `json:"latencyExecute"`
		LatencyTotalMean                                         int            `json:"latencyTotal_mean"`
		LatencyTotal                                             map[string]int `json:"latencyTotal"`
		PropertyCircuitBreakerRequestVolumeThreshold             int            `json:"propertyValue_circuitBreakerRequestVolumeThreshold"`
		PropertyCircuitbreakerSleepWindowInMilliseconds          int            `json:"propertyValue_circuitBreakerSleepWindowInMilliseconds"`
		PropertyCircuitBreakerErrorThresholdPercentage           int            `json:"propertyValue_circuitBreakerErrorThresholdPercentage"`
		PropertyCircuitBreakerForceOpen                          bool           `json:"propertyValue_circuitBreakerForceOpen"`
		PropertyCircuitBreakerForceClosed                        bool           `json:"propertyValue_circuitBreakerForceClosed"`
		PropertyExecutionIsolationStrategy                       string         `json:"propertyValue_executionIsolationStrategy"`
		ProperyExecutionIsolationThreadTimeoutInMilliseconds     int            `json:"propertyValue_executionIsolationThreadTimeoutInMilliseconds"`
		PropertyExecutionIsolationThreadInterruptOnTimeout       bool           `json:"propertyValue_executionIsolationThreadInterruptOnTimeout"`
		PropertyExecutionIsolationSemaphoreMaxConcurrentRequests int            `json:"propertyValue_executionIsolationSemaphoreMaxConcurrentRequests"`
		PropertyFallbackIsolationSemaphoreMaxConcurrentRequests  int            `json:"propertyValue_fallbackIsolationSemaphoreMaxConcurrentRequests"`
		PropertyRequestCacheEnabled                              bool           `json:"propertyValue_requestCacheEnabled"`
		PropertyRequestLogEnabled                                bool           `json:"propertyValue_requestLogEnabled"`
		PropertyMetricsRollingStatisticalWindowInMilliseconds    int            `json:"propertyValue_metricsRollingStatisticalWindowInMilliseconds"`
		ReportingHosts                                           int            `json:"reportingHosts"`
	}{
		"HystrixCommand",
		m.CommandName(),
		m.CommandName(),
		int(time.Now().UnixNano() / 1000000),
		h.executor.IsCircuitBreakerOpen(m.CommandName()),
		m.ErrorPercentage(),
		m.ErrorCount(),
		m.TotalRequests(),
		0,
		0,
		m.RollingSum(requestlog.Failure),
		m.RollingSum(requestlog.FallbackFailure),
		0,
		m.RollingSum(requestlog.FallbackSuccess),
		m.RollingSum(requestlog.ResponseFromCache),
		0,
		m.RollingSum(requestlog.ShortCircuited),
		m.RollingSum(requestlog.Success),
		m.RollingSum(requestlog.SemaphoreRejected),
		m.RollingSum(requestlog.Timeout),
		0,
		toMilliseconds(m.ExecutionTimeMean()),
		collectExecutionPercentiles(m),
		toMilliseconds(m.ExecutionTimeMean()),
		collectExecutionPercentiles(m),
		20,
		5000,
		50,
		false,
		false,
		"THREAD",
		1000,
		true,
		10,
		10,
		true,
		true,
		10000,
		1,
	}
	e.Encode(&metrics)
}

func collectExecutionPercentiles(m *metrics.CommandMetrics) map[string]int {
	ps := make(map[string]int)
	ps["0"] = toMilliseconds(m.ExecutionTimePercentile(0))
	ps["25"] = toMilliseconds(m.ExecutionTimePercentile(25))
	ps["50"] = toMilliseconds(m.ExecutionTimePercentile(50))
	ps["75"] = toMilliseconds(m.ExecutionTimePercentile(75))
	ps["90"] = toMilliseconds(m.ExecutionTimePercentile(90))
	ps["95"] = toMilliseconds(m.ExecutionTimePercentile(95))
	ps["99"] = toMilliseconds(m.ExecutionTimePercentile(99))
	ps["99.5"] = toMilliseconds(m.ExecutionTimePercentile(99.5))
	ps["100"] = toMilliseconds(m.ExecutionTimePercentile(100))
	return ps
}

func toMilliseconds(d time.Duration) int {
	return int(d / time.Millisecond)
}
