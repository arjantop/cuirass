package metrics

import (
	"sync"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
)

type CommandMetrics struct {
	name          string
	eventCounters map[requestlog.ExecutionEvent]*num.RollingNumber
	clock         util.Clock
	lock          *sync.RWMutex
}

func newCommandMetrics(clock util.Clock, name string) *CommandMetrics {
	return &CommandMetrics{
		name:          name,
		eventCounters: make(map[requestlog.ExecutionEvent]*num.RollingNumber),
		clock:         clock,
		lock:          new(sync.RWMutex),
	}
}

func newRollingNumber(clock util.Clock) *num.RollingNumber {
	return num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets, clock)
}

func (m *CommandMetrics) CommandName() string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.name
}

func (m *CommandMetrics) TotalRequests() int {
	successCount := m.RollingSum(requestlog.Success)
	failureCount := m.RollingSum(requestlog.Failure)
	timeoutCount := m.RollingSum(requestlog.Timeout)
	shortCircuitedCount := m.RollingSum(requestlog.ShortCircuited)
	return successCount + failureCount + shortCircuitedCount + timeoutCount
}

func (m *CommandMetrics) ErrorCount() int {
	failureCount := m.RollingSum(requestlog.Failure)
	timeoutCount := m.RollingSum(requestlog.Timeout)
	shortCircuitedCount := m.RollingSum(requestlog.ShortCircuited)
	return failureCount + shortCircuitedCount + timeoutCount
}

func (m *CommandMetrics) ErrorPercentage() int {
	total := m.TotalRequests()
	if total == 0 {
		return 0
	}
	return int(float64(m.ErrorCount()) / float64(total) * 100)
}

func (m *CommandMetrics) incEventCount(e requestlog.ExecutionEvent) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.eventCounters[e]; ok {
		c.Increment()
	} else {
		c := newRollingNumber(m.clock)
		c.Increment()
		m.eventCounters[e] = c
	}
}

func (m *CommandMetrics) RollingSum(e requestlog.ExecutionEvent) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.eventCounters[e]; ok {
		return int(c.Sum())
	}
	return 0
}

type ExecutionMetrics struct {
	clock          util.Clock
	commandMetrics map[string]*CommandMetrics
	lock           *sync.RWMutex
}

func NewExecutionMetrics(clock util.Clock) *ExecutionMetrics {
	return &ExecutionMetrics{
		clock:          clock,
		commandMetrics: make(map[string]*CommandMetrics),
		lock:           new(sync.RWMutex),
	}
}

func (m *ExecutionMetrics) All() []*CommandMetrics {
	m.lock.RLock()
	defer m.lock.RUnlock()
	c := make([]*CommandMetrics, 0, len(m.commandMetrics))
	for _, m := range m.commandMetrics {
		c = append(c, m)
	}
	return c
}

func (m *ExecutionMetrics) ForCommand(name string) *CommandMetrics {
	return m.fetchMetrics(name)
}

func (m *ExecutionMetrics) Update(name string, evs ...requestlog.ExecutionEvent) {
	metrics := m.fetchMetrics(name)
	for _, e := range evs {
		metrics.incEventCount(e)
	}
}

func (m *ExecutionMetrics) fetchMetrics(name string) *CommandMetrics {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m, ok := m.commandMetrics[name]; ok {
		return m
	}
	metrics := newCommandMetrics(m.clock, name)
	m.commandMetrics[name] = metrics
	return metrics
}
