package cuirass

import (
	"errors"
	"sync"
	"time"

	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/cuirass/metrics"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
	"golang.org/x/net/context"
)

var (
	UnknownPanic      = errors.New("unknown panic")
	SemaphoreRejected = errors.New("semaphore rejected")
)

// Executor is a main service that knows how to execute commands and handle
// their errors.
// Executor must be safe to be accessed by multiple goroutines.
type Executor interface {
	Exec(ctx context.Context, cmd *Command) (result interface{}, err error)
}

// CommandExecutor is an implementation of an Executor interface.
type CommandExecutor struct {
	clock           util.Clock
	cfg             vaquita.DynamicConfig
	circuitBreakers cbMap
	semaphores      *SemaphoreFactory
	metrics         *metrics.ExecutionMetrics
}

// NewExecutor constructs a new empty executor.
func NewExecutor(cfg vaquita.DynamicConfig) *CommandExecutor {
	return NewExecutorWithClock(cfg, util.NewClock())
}

func NewExecutorWithClock(cfg vaquita.DynamicConfig, clock util.Clock) *CommandExecutor {
	return &CommandExecutor{
		clock:           clock,
		cfg:             cfg,
		circuitBreakers: newCbMap(),
		semaphores:      NewSemaphoreFactory(),
		metrics:         metrics.NewExecutionMetrics(metrics.NewMetricsProperties(cfg), clock),
	}
}

func (e *CommandExecutor) Metrics() *metrics.ExecutionMetrics {
	return e.metrics
}

func (e *CommandExecutor) IsCircuitBreakerOpen(cmdName string) bool {
	if cb, ok := e.circuitBreakers.get(cmdName); ok {
		return cb.IsOpen()
	}
	return false
}

// Exec executes a command and handles command execution errors.
// If command fails with an error or panics Fallback function with fallback logic
// is executed. Every command execution is guarded by an internal circuit-breaker.
// Panics are recovered and returned as errors.
func (e *CommandExecutor) Exec(ctx context.Context, cmd *Command) (result interface{}, err error) {
	var responseFromCache bool
	stats := newExecutionStats(time.Now())
	defer func() {
		if r := recover(); r != nil {
			if !cmd.Properties(e.cfg).FallbackEnabled.Get() {
				stats.addEvent(requestlog.Failure)
				e.logRequest(ctx, stats.toExecutionInfo(cmd.Name()), cmd.Properties(e.cfg))
				return
			} else {
				result, err = e.execFallback(ctx, cmd, stats, r)
			}
		} else if !responseFromCache {
			// The request was successfully completed.
			stats.addEvent(requestlog.Success)
			e.logRequest(ctx, stats.toExecutionInfo(cmd.Name()), cmd.Properties(e.cfg))
		}
		if cache := e.getRequestCache(ctx, cmd); cache != nil && !responseFromCache {
			cache.Add(cmd.Name(), cmd.CacheKey(), stats.toExecutionInfo(cmd.Name()), result, err)
		}
	}()

	if cache := e.getRequestCache(ctx, cmd); cache != nil {
		if ec := cache.Get(cmd.Name(), cmd.CacheKey()); ec != nil {
			// Return the cached return values straight from cache.
			result, err = ec.Response()
			// Mark that the response came from cache and we already did the logging.
			responseFromCache = true
			e.logRequest(ctx, *ec.ExecutionInfo(), cmd.Properties(e.cfg))
			return
		}
	}

	if timeout := cmd.Properties(e.cfg).ExecutionTimeout.Get(); timeout != 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, cmd.Properties(e.cfg).ExecutionTimeout.Get())
		defer cancel()
	}
	cb := e.getCircuitBreakerForCommand(cmd)
	// Execute the command in the context of its circuit-breaker.
	err = cb.Do(func() error {
		s := e.semaphores.Get(cmd.Group(), cmd.Properties(e.cfg).ExecutionMaxConcurrentRequests.Get())
		if ok := s.TryAcquire(); ok {
			defer s.Release()
			rr, rerr := cmd.Run(ctx)
			result = rr
			return rerr
		}
		return SemaphoreRejected
	})
	if err != nil {
		// Panic with error and handle it the same as panic.
		panic(err)
	}
	return
}

func (e *CommandExecutor) getRequestCache(ctx context.Context, cmd *Command) *requestcache.RequestCache {
	cache := requestcache.FromContext(ctx)
	if cache != nil && cmd.IsCacheable() && cmd.Properties(e.cfg).RequestCacheEnabled.Get() {
		return cache
	}
	return nil
}

// executionStats holds the execution start time and the events that occurred
// during command execution.
type executionStats struct {
	startTime time.Time
	events    []requestlog.ExecutionEvent
}

// newExecutionStats constructs a new executionStats with execution start time
// set to startTime.
func newExecutionStats(startTime time.Time) executionStats {
	return executionStats{
		startTime: startTime,
		events:    make([]requestlog.ExecutionEvent, 0),
	}
}

// addEvent adds an event that occurred to the event log.
func (e *executionStats) addEvent(event requestlog.ExecutionEvent) {
	e.events = append(e.events, event)
}

// toExecutionInfo constructs an ExecutionInfo from the gathered data.
func (e *executionStats) toExecutionInfo(commandName string) requestlog.ExecutionInfo {
	return requestlog.NewExecutionInfo(commandName, time.Since(e.startTime), e.events)
}

// logRequest logs a request if the context contains a RequestLogger.
func (e *CommandExecutor) logRequest(ctx context.Context, info requestlog.ExecutionInfo, props *CommandProperties) {
	e.metrics.Update(info.CommandName(), info.ExecutionTime(), info.Events()...)
	if logger := requestlog.FromContext(ctx); props.RequestLogEnabled.Get() && logger != nil {
		logger.AddExecutionInfo(info)
	}
}

// executeFallback handles a fallback for a failed command.
// Because a Fallback can panic too errors are recovered the same way as for Exec.
func (e *CommandExecutor) execFallback(
	ctx context.Context,
	cmd *Command,
	stats executionStats,
	r interface{}) (result interface{}, err error) {

	defer func() {
		if r := recover(); r != nil {
			if err != FallbackNotImplemented {
				// If the fallback is not implemented we don't want to log the failure.
				stats.addEvent(requestlog.FallbackFailure)
			}
			err = panicToError(r)
		}
		e.logRequest(ctx, stats.toExecutionInfo(cmd.Name()), cmd.Properties(e.cfg))
	}()

	addEventForRequest(&stats, r)

	result, err = cmd.Fallback(ctx)
	if err != nil {
		panic(r)
	}
	stats.addEvent(requestlog.FallbackSuccess)
	return
}

// Add the event for executed command failure to the log.
func addEventForRequest(stats *executionStats, r interface{}) {
	switch x := r.(type) {
	case error:
		if x == context.DeadlineExceeded {
			stats.addEvent(requestlog.Timeout)
		} else if x == circuitbreaker.CircuitOpenError {
			stats.addEvent(requestlog.ShortCircuited)
		} else if x == SemaphoreRejected {
			stats.addEvent(requestlog.SemaphoreRejected)
		} else {
			stats.addEvent(requestlog.Failure)
		}
	default:
		stats.addEvent(requestlog.Failure)
	}
}

// panicToError converts a panic value to a matching error value or a generic
// UnknownPanic for unhandled types.
func panicToError(r interface{}) (err error) {
	switch x := r.(type) {
	case error:
		err = x
	case string:
		err = errors.New(x)
	default:
		err = UnknownPanic
	}
	return
}

// getcircuitbreakerforcommand returns a circuit breaker for a command or constructs
// a new one and returns it.
func (e *CommandExecutor) getCircuitBreakerForCommand(cmd *Command) *circuitbreaker.CircuitBreaker {
	if cb, ok := e.circuitBreakers.get(cmd.Name()); ok {
		return cb
	} else {
		cb := circuitbreaker.New(
			cmd.Properties(e.cfg).CircuitBreaker,
			e.metrics.Properties().HealthSnapshotInterval,
			e.clock)
		e.circuitBreakers.set(cmd.Name(), cb)
		return cb
	}
}

// cmMap is a simple map wrapper for safe concurrent access.
// Because most of the operations are just reading it is simply guarded by
// one RWMutex lock.
type cbMap struct {
	values map[string]*circuitbreaker.CircuitBreaker
	lock   *sync.RWMutex
}

// newCbMap constructs a new empty cmMap.
func newCbMap() cbMap {
	return cbMap{
		values: make(map[string]*circuitbreaker.CircuitBreaker),
		lock:   new(sync.RWMutex),
	}
}

// get returns a circuit breaker for a command with a given name.
// Method is safe to access by multiple readers.
func (m *cbMap) get(name string) (*circuitbreaker.CircuitBreaker, bool) {
	m.lock.RLock()
	cb, ok := m.values[name]
	m.lock.RUnlock()
	return cb, ok
}

// set adds a circuit breaker to the map for a given command name.
// Only one writer and no readers can access the map when executing set.
func (m *cbMap) set(name string, cb *circuitbreaker.CircuitBreaker) {
	m.lock.Lock()
	m.values[name] = cb
	m.lock.Unlock()
}
