package metrics

import (
	"sync"
	"time"

	"github.com/arjantop/cuirass/num"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
)

const (
	RollingPercentileBucketSizeDefault = 100
	HealthSnapshotIntervalDefault      = 500 * time.Millisecond
)

type MetricsProperties struct {
	RollingPercentileBucketSize vaquita.IntProperty
	HealthSnapshotInterval      vaquita.DurationProperty
}

func NewMetricsProperties(cfg vaquita.DynamicConfig) *MetricsProperties {
	f := vaquita.NewPropertyFactory(cfg)
	return &MetricsProperties{
		RollingPercentileBucketSize: f.GetIntProperty("metrics.rollingPercentile.bucketSize", RollingPercentileBucketSizeDefault),
		HealthSnapshotInterval:      f.GetDurationProperty("metrics.healthSnapshot.intervalInMilliseconds", HealthSnapshotIntervalDefault, time.Millisecond),
	}
}

type CommandMetrics struct {
	name          string
	eventCounters map[requestlog.ExecutionEvent]*num.RollingNumber
	executionTime *num.RollingPercentile
	clock         util.Clock
	lock          *sync.RWMutex
}

func newCommandMetrics(props *MetricsProperties, clock util.Clock, name string) *CommandMetrics {
	return &CommandMetrics{
		name:          name,
		eventCounters: make(map[requestlog.ExecutionEvent]*num.RollingNumber),
		executionTime: num.NewRollingPercentile(num.DefaultWindowSize, num.DefaultWindowBuckets, props.RollingPercentileBucketSize.Get(), clock),
		clock:         clock,
		lock:          new(sync.RWMutex),
	}
}

func newRollingNumber(clock util.Clock) *num.RollingNumber {
	return num.NewRollingNumber(num.DefaultWindowSize, num.DefaultWindowBuckets, clock)
}

func (m *CommandMetrics) CommandName() string {
	m.lock.RLock()
	n := m.name
	m.lock.RUnlock()
	return n
}

func (m *CommandMetrics) TotalRequests() int {
	successCount := m.RollingSum(requestlog.Success)
	failureCount := m.RollingSum(requestlog.Failure)
	timeoutCount := m.RollingSum(requestlog.Timeout)
	shortCircuitedCount := m.RollingSum(requestlog.ShortCircuited)
	semaphoreRejected := m.RollingSum(requestlog.SemaphoreRejected)
	return successCount + failureCount + shortCircuitedCount + timeoutCount + semaphoreRejected
}

func (m *CommandMetrics) ErrorCount() int {
	failureCount := m.RollingSum(requestlog.Failure)
	timeoutCount := m.RollingSum(requestlog.Timeout)
	shortCircuitedCount := m.RollingSum(requestlog.ShortCircuited)
	semaphoreRejected := m.RollingSum(requestlog.SemaphoreRejected)
	return failureCount + shortCircuitedCount + timeoutCount + semaphoreRejected
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
	if hasEvent(evs, requestlog.ResponseFromCache) {
		m.findEventCounter(requestlog.ResponseFromCache).Increment()
	} else {
		for _, e := range evs {
			m.findEventCounter(e).Increment()
		}
		if !hasEvent(evs, requestlog.ShortCircuited) && !hasEvent(evs, requestlog.SemaphoreRejected) {
			m.executionTime.Add(int(executionTime))
		}
	}
	m.lock.Unlock()
}

func (m *CommandMetrics) findEventCounter(e requestlog.ExecutionEvent) *num.RollingNumber {
	c, ok := m.eventCounters[e]
	if !ok {
		c = newRollingNumber(m.clock)
		m.eventCounters[e] = c
	}
	return c
}

func hasEvent(events []requestlog.ExecutionEvent, e requestlog.ExecutionEvent) bool {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i] == e {
			return true
		}
	}
	return false
}

func (m *CommandMetrics) RollingSum(e requestlog.ExecutionEvent) int {
	m.lock.Lock()
	if c, ok := m.eventCounters[e]; ok {
		m.lock.Unlock()
		return int(c.Sum())
	}
	m.lock.Unlock()
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
	props          *MetricsProperties
	commandMetrics map[string]*CommandMetrics
	lock           *sync.RWMutex
}

func NewExecutionMetrics(props *MetricsProperties, clock util.Clock) *ExecutionMetrics {
	return &ExecutionMetrics{
		clock:          clock,
		props:          props,
		commandMetrics: make(map[string]*CommandMetrics),
		lock:           new(sync.RWMutex),
	}
}

func (m *ExecutionMetrics) Properties() *MetricsProperties {
	m.lock.RLock()
	p := m.props
	m.lock.RUnlock()
	return p
}

func (m *ExecutionMetrics) All() []*CommandMetrics {
	m.lock.RLock()
	c := make([]*CommandMetrics, 0, len(m.commandMetrics))
	for _, m := range m.commandMetrics {
		c = append(c, m)
	}
	m.lock.RUnlock()
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
	metrics := newCommandMetrics(m.props, m.clock, name)
	m.commandMetrics[name] = metrics
	return metrics
}
