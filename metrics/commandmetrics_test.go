package metrics_test

import (
	"testing"
	"time"

	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
	"github.com/stretchr/testify/assert"
)

func newTestingExecutionMetrics() *metrics.ExecutionMetrics {
	cfg := vaquita.NewEmptyMapConfig()
	f := vaquita.NewPropertyFactory(cfg)
	return metrics.NewExecutionMetrics(&metrics.MetricsProperties{
		RollingPercentileBucketSize: f.GetIntProperty("percentileBucketSize", 100),
	}, util.NewClock())
}

func TestCommandMetricsCommandName(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("cmd")
	assert.Equal(t, "cmd", m.CommandName())
}

func TestCommandMetricsRequestCount(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("cmd1")
	assert.Equal(t, 0, m.TotalRequests())
	assert.Equal(t, 0, m.ErrorCount())

	addEventAndAssertCount(t, em, "cmd1", requestlog.Success, 1, 0)
	addEventAndAssertCount(t, em, "cmd1", requestlog.Failure, 2, 1)
	addEventAndAssertCount(t, em, "cmd1", requestlog.Timeout, 3, 2)
	addEventAndAssertCount(t, em, "cmd1", requestlog.ShortCircuited, 4, 3)
	addEventAndAssertCount(t, em, "cmd1", requestlog.Success, 5, 3)
}

func TestCommandMetricsSeparatePerCommand(t *testing.T) {
	em := newTestingExecutionMetrics()

	addEventAndAssertCount(t, em, "cmd1", requestlog.Success, 1, 0)
	addEventAndAssertCount(t, em, "cmd2", requestlog.Failure, 1, 1)
	addEventAndAssertCount(t, em, "cmd1", requestlog.Success, 2, 0)
}

func addEventAndAssertCount(t *testing.T, em *metrics.ExecutionMetrics, name string, e requestlog.ExecutionEvent, rc, ec int) {
	em.Update(name, 0, e)
	m := em.ForCommand(name)
	assert.Equal(t, rc, m.TotalRequests())
	assert.Equal(t, ec, m.ErrorCount())
}

func TestCommandMetricsErrorPercentage(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("cmd1")
	assert.Equal(t, 0, m.ErrorPercentage())

	em.Update("cmd1", 0, requestlog.Success)
	em.Update("cmd1", 0, requestlog.Failure)
	em.Update("cmd1", 0, requestlog.Timeout)
	em.Update("cmd1", 0, requestlog.Failure)
	em.Update("cmd1", 0, requestlog.ShortCircuited)

	assert.Equal(t, 80, m.ErrorPercentage())

	em.Update("cmd1", 0, requestlog.Success)
	assert.Equal(t, 66, m.ErrorPercentage())

	em.Update("cmd1", 0, requestlog.Success)
	em.Update("cmd1", 0, requestlog.Success)
	assert.Equal(t, 50, m.ErrorPercentage())
}

func TestCommandMetricsRollingSum(t *testing.T) {
	em := newTestingExecutionMetrics()

	addEventAndAssertRollingSum(t, em, "c", requestlog.Success, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.Failure, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.Timeout, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.ShortCircuited, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.FallbackSuccess, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.FallbackFailure, 1)
	addEventAndAssertRollingSum(t, em, "c", requestlog.ResponseFromCache, 1)

	addEventAndAssertRollingSum(t, em, "c", requestlog.Success, 2)
	addEventAndAssertRollingSum(t, em, "c", requestlog.Success, 3)
	addEventAndAssertRollingSum(t, em, "c", requestlog.Failure, 2)
}

func addEventAndAssertRollingSum(t *testing.T, em *metrics.ExecutionMetrics, name string, e requestlog.ExecutionEvent, expected int) {
	em.Update(name, 0, e)
	m := em.ForCommand(name)
	assert.Equal(t, expected, m.RollingSum(e))
}

func TestExecutionMetricsMultipleEvents(t *testing.T) {
	em := newTestingExecutionMetrics()
	em.Update("command", 0, requestlog.Failure, requestlog.FallbackSuccess)
	m := em.ForCommand("command")
	assert.Equal(t, 1, m.RollingSum(requestlog.Failure))
	assert.Equal(t, 1, m.RollingSum(requestlog.FallbackSuccess))
}

func TestExecutionMetricsExecutionTimePercentile(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("command")
	em.Update("command", 5*time.Millisecond, requestlog.Failure, requestlog.FallbackSuccess)
	assert.Equal(t, 5*time.Millisecond, m.ExecutionTimePercentile(100))
	assert.Equal(t, 5*time.Millisecond, m.ExecutionTimeMean())
	em.Update("command", 10*time.Millisecond, requestlog.Failure, requestlog.FallbackSuccess)
	assert.Equal(t, 10*time.Millisecond, m.ExecutionTimePercentile(100))
	assert.Equal(t, 7500*time.Microsecond, m.ExecutionTimeMean())
}

func TestExecutionMetricsIgoreExecutionTimeOfEvents(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("command")
	em.Update("command", 10, requestlog.ResponseFromCache)
	assert.Equal(t, 0, m.ExecutionTimePercentile(100))
	em.Update("command", 20, requestlog.ShortCircuited)
	assert.Equal(t, 0, m.ExecutionTimePercentile(100))
	em.Update("command", 20, requestlog.SemaphoreRejected)
	assert.Equal(t, 0, m.ExecutionTimePercentile(100))
}

func TestExecutionMetricsIgoreEventsWhenResponseFromCache(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("command")
	em.Update("command", 0, requestlog.Failure, requestlog.FallbackSuccess, requestlog.ResponseFromCache)
	assert.Equal(t, 0, m.RollingSum(requestlog.Failure))
	assert.Equal(t, 0, m.RollingSum(requestlog.FallbackSuccess))
	assert.Equal(t, 1, m.RollingSum(requestlog.ResponseFromCache))
}
