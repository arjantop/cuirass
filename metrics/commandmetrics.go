package metrics

import (
	"sync"
	"time"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
)

type CommandMetrics struct {
	name          string
	eventCounters map[requestlog.ExecutionEvent]*num.RollingNumber
	executionTime *num.RollingPercentile
	clock         util.Clock
	lock          *sync.RWMutex
}

func newCommandMetrics(clock util.Clock, name string) *CommandMetrics {
	return &CommandMetrics{
		name:          name,
		eventCounters: make(map[requestlog.ExecutionEvent]*num.RollingNumber),
		executionTime: num.NewRollingPercentile(num.DefaultWindowSize, num.DefaultWindowBuckets, clock),
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

func (m *CommandMetrics) update(executionTime time.Duration, evs ...requestlog.ExecutionEvent) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !isResponseFromCache(evs) {
		for _, e := range evs {
			m.findEventCounter(e).Increment()
		}
		m.executionTime.Add(int(executionTime))
	} else {
		m.findEventCounter(requestlog.ResponseFromCache).Increment()
	}
}

func (m *CommandMetrics) findEventCounter(e requestlog.ExecutionEvent) *num.RollingNumber {
	c, ok := m.eventCounters[e]
	if !ok {
		c = newRollingNumber(m.clock)
		m.eventCounters[e] = c
	}
	return c
}

func isResponseFromCache(events []requestlog.ExecutionEvent) bool {
	return events[len(events)-1] == requestlog.ResponseFromCache
}

func (m *CommandMetrics) RollingSum(e requestlog.ExecutionEvent) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok := m.eventCounters[e]; ok {
		return int(c.Sum())
	}
	return 0
}

func (m *CommandMetrics) ExecutionTimeMean() time.Duration {
	return time.Duration(m.executionTime.Mean())
}

func (m *CommandMetrics) ExecutionTimePercentile(p float64) time.Duration {
	return time.Duration(m.executionTime.Get(p))
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

func (m *ExecutionMetrics) Update(name string, executionTime time.Duration, evs ...requestlog.ExecutionEvent) {
	metrics := m.fetchMetrics(name)
	metrics.update(executionTime, evs...)
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
