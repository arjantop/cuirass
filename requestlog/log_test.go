package requestlog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddRequestMany(t *testing.T) {
	logger := newRequestLog()
	commands := []string{"Foo", "Bar", "Baz"}
	logger.AddExecutionInfo(NewExecutionInfo(commands[0], 2*time.Second, []ExecutionEvent{Timeout}))
	assert.Equal(t, "Foo[TIMEOUT][2000ms]", logger.String())
	logger.AddExecutionInfo(NewExecutionInfo(commands[1], 10*time.Nanosecond, []ExecutionEvent{Success}))
	logger.AddExecutionInfo(NewExecutionInfo(commands[2], 210*time.Millisecond, []ExecutionEvent{Failure}))
	assert.Equal(t, "Foo[TIMEOUT][2000ms], Bar[SUCCESS][0ms], Baz[FAILURE][210ms]", logger.String())
}

func TestRequestStringCollapsing(t *testing.T) {
	logger := newRequestLog()
	commands := []string{"Foo", "Bar", "Baz"}
	logger.AddExecutionInfo(NewExecutionInfo(commands[0], 2*time.Second, []ExecutionEvent{Timeout}))
	logger.AddExecutionInfo(NewExecutionInfo(commands[0], 1*time.Second, []ExecutionEvent{Success}))
	// Same commands with different events are not collapsed.
	assert.Equal(t, "Foo[TIMEOUT][2000ms], Foo[SUCCESS][1000ms]", logger.String())

	logger2 := newRequestLog()
	logger2.AddExecutionInfo(NewExecutionInfo(commands[1], 10*time.Millisecond, []ExecutionEvent{Success}))
	logger2.AddExecutionInfo(NewExecutionInfo(commands[0], 11*time.Millisecond, []ExecutionEvent{Success}))
	logger2.AddExecutionInfo(NewExecutionInfo(commands[0], 2*time.Millisecond, []ExecutionEvent{Success}))
	// All commands executions will be aggregated, not just consecutive ones.
	logger2.AddExecutionInfo(NewExecutionInfo(commands[2], 1*time.Millisecond, []ExecutionEvent{Success}))
	logger2.AddExecutionInfo(NewExecutionInfo(commands[0], 8*time.Millisecond, []ExecutionEvent{Success}))
	// Execution times of collapsed commands are summed.
	assert.Equal(t, "Bar[SUCCESS][10ms], Foo[SUCCESS][21ms]x3, Baz[SUCCESS][1ms]", logger2.String())
}

func TestStringMultipleEvents(t *testing.T) {
	logger := newRequestLog()
	logger.AddExecutionInfo(NewExecutionInfo("Foo", 0, []ExecutionEvent{Timeout, FallbackSuccess, ResponseFromCache}))
	assert.Equal(t, "Foo[TIMEOUT, FALLBACK_SUCCESS, RESPONSE_FROM_CACHE][0ms]", logger.String())

	logger2 := newRequestLog()
	logger2.AddExecutionInfo(NewExecutionInfo("Foo", 0, []ExecutionEvent{ShortCircuited, FallbackFailure}))
	assert.Equal(t, "Foo[SHORT_CIRCUITED, FALLBACK_FAILURE][0ms]", logger2.String())
}

func TestLastRequest(t *testing.T) {
	logger := newRequestLog()
	info1 := NewExecutionInfo("Foo", 1, []ExecutionEvent{Success})
	logger.AddExecutionInfo(info1)
	assert.Equal(t, info1, *logger.LastRequest())
	info2 := NewExecutionInfo("Bar", 2, []ExecutionEvent{Success})
	logger.AddExecutionInfo(info2)
	assert.Equal(t, info2, *logger.LastRequest())
}

func TestExecutionInfoExecutionTimeNegative(t *testing.T) {
	info := NewExecutionInfo("Foo", -10*time.Millisecond, []ExecutionEvent{Success})
	assert.Equal(t, 0, info.ExecutionTime())
}

func TestStringEmptyRequestLog(t *testing.T) {
	logger := newRequestLog()
	assert.Equal(t, "", logger.String())
}

func TestStringEvents(t *testing.T) {
	logger := newRequestLog()
	logger.AddExecutionInfo(NewExecutionInfo("Foo", 0, []ExecutionEvent{SemaphoreRejected}))
	assert.Equal(t, "Foo[SEMAPHORE_REJECTED][0ms]", logger.String())
}
