package metrics_test

import (
	"testing"

	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/stretchr/testify/assert"
)

func newTestingExecutionMetrics() *metrics.ExecutionMetrics {
	return metrics.NewExecutionMetrics(util.NewClock())
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
	em.Update(name, e)
	m := em.ForCommand(name)
	assert.Equal(t, rc, m.TotalRequests())
	assert.Equal(t, ec, m.ErrorCount())
}

func TestCommandMetricsErrorPercentage(t *testing.T) {
	em := newTestingExecutionMetrics()
	m := em.ForCommand("cmd1")
	assert.Equal(t, 0, m.ErrorPercentage())

	em.Update("cmd1", requestlog.Success)
	em.Update("cmd1", requestlog.Failure)
	em.Update("cmd1", requestlog.Timeout)
	em.Update("cmd1", requestlog.Failure)
	em.Update("cmd1", requestlog.ShortCircuited)

	assert.Equal(t, 80, m.ErrorPercentage())

	em.Update("cmd1", requestlog.Success)
	assert.Equal(t, 66, m.ErrorPercentage())

	em.Update("cmd1", requestlog.Success)
	em.Update("cmd1", requestlog.Success)
	assert.Equal(t, 50, m.ErrorPercentage())
}
