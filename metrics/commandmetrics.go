package metrics

import (
	"sync"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
)

type CommandMetrics struct {
	name          string
	totalRequests *num.RollingNumber
	errorCount    *num.RollingNumber
	lock          *sync.RWMutex
}

func newCommandMetrics(clock util.Clock, name string) *CommandMetrics {
	return &CommandMetrics{
		name:          name,
		totalRequests: newRollingNumber(clock),
		errorCount:    newRollingNumber(clock),
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

func (m *CommandMetrics) incTotalRequests() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.totalRequests.Increment()
}

func (m *CommandMetrics) TotalRequests() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return int(m.totalRequests.Sum())
}

func (m *CommandMetrics) incErrorCount() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.errorCount.Increment()
}

func (m *CommandMetrics) ErrorCount() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return int(m.errorCount.Sum())
}

func (m *CommandMetrics) ErrorPercentage() int {
	if m.TotalRequests() == 0 {
		return 0
	}
	return int(float64(m.ErrorCount()) / float64(m.TotalRequests()) * 100)
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

func (m *ExecutionMetrics) Update(name string, result requestlog.ExecutionEvent) {
	metrics := m.fetchMetrics(name)
	if result != requestlog.Success {
		metrics.incErrorCount()
	}
	metrics.incTotalRequests()
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
