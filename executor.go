package cuirass

import (
	"errors"
	"sync"
	"time"

	"code.google.com/p/go.net/context"

	"github.com/arjantop/cuirass/circuitbreaker"
	"github.com/arjantop/cuirass/requestcache"
	"github.com/arjantop/cuirass/requestlog"
	"github.com/arjantop/cuirass/util"
	"github.com/arjantop/vaquita"
)

var UnknownPanic = errors.New("Unknown panic")

var DefaultRequestTimeout time.Duration = time.Second

// Executor is a main service that knows how to execute commands and handle
// their errors.
// Executor is safe to be accessed by multiple threads.
type Executor struct {
	cfg             vaquita.DynamicConfig
	circuitBreakers cbMap
}

// NewExecutor constructs a new empty executor.
func NewExecutor(cfg vaquita.DynamicConfig) *Executor {
	return &Executor{
		cfg:             cfg,
		circuitBreakers: newCbMap(),
	}
}

// Exec executes a command and handles command execution errors.
// If command fails with an error or panics Fallback function with fallback logic
// is executed. Every command execution is guarded by an internal circuit-breaker.
// Panics are recovered and returned as errors.
func (e *Executor) Exec(ctx context.Context, cmd *Command) (result interface{}, err error) {
	var responseFromCache bool
	stats := newExecutionStats(time.Now())
	defer func() {
		if r := recover(); r != nil {
			if !cmd.Properties(e.cfg).FallbackEnabled.Get() {
				stats.addEvent(requestlog.Failure)
				logRequest(ctx, stats.toExecutionInfo(cmd.Name()))
				return
			} else {
				result, err = execFallback(ctx, cmd, stats, r)
			}
		} else if !responseFromCache {
			// The request was successfully completed.
			stats.addEvent(requestlog.Success)
			logRequest(ctx, stats.toExecutionInfo(cmd.Name()))
		}
		if cache := requestcache.FromContext(ctx); !responseFromCache && cmd.IsCacheable() && cache != nil {
			cache.Add(cmd.Name(), cmd.CacheKey(), stats.toExecutionInfo(cmd.Name()), result, err)
		}
	}()

	if cache := requestcache.FromContext(ctx); cmd.IsCacheable() && cache != nil {
		if ec := cache.Get(cmd.Name(), cmd.CacheKey()); ec != nil {
			// Return the cached return values straight from cache.
			result, err = ec.Response()
			// Mark that the response came from cache and we already did the logging.
			responseFromCache = true
			logRequest(ctx, ec.ExecutionInfo())
			return
		}
	}

	ctx, cancel := context.WithTimeout(ctx, cmd.Properties(e.cfg).ExecutionTimeout.Get())
	defer cancel()
	cb := e.getCircuitBreakerForCommand(cmd)
	// Execute the command in the context of its circuit-breaker.
	err = cb.Do(func() error {
		rr, rerr := cmd.Run(ctx)
		result = rr
		return rerr
	})
	if err != nil {
		// Panic with error and handle it the same as panic.
		panic(err)
	}
	return
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
func (e *executionStats) toExecutionInfo(commandName string) *requestlog.ExecutionInfo {
	return requestlog.NewExecutionInfo(commandName, time.Since(e.startTime), e.events)
}

// logRequest logs a request if the context contains a RequestLogger.
func logRequest(ctx context.Context, info *requestlog.ExecutionInfo) {
	if logger := requestlog.FromContext(ctx); logger != nil {
		logger.AddExecutionInfo(info)
	}
}

// executeFallback handles a fallback for a failed command.
// Because a Fallback can panic too errors are recovered the same way as for Exec.
func execFallback(
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
		logRequest(ctx, stats.toExecutionInfo(cmd.Name()))
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
func (e *Executor) getCircuitBreakerForCommand(cmd *Command) *circuitbreaker.CircuitBreaker {
	if cb, ok := e.circuitBreakers.get(cmd.Name()); ok {
		return cb
	} else {
		cb := circuitbreaker.New(cmd.Properties(e.cfg).CircuitBreaker, util.NewClock())
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
	defer m.lock.RUnlock()
	cb, ok := m.values[name]
	return cb, ok
}

// set adds a circuit breaker to the map for a given command name.
// Only one writer and no readers can access the map when executing set.
func (m *cbMap) set(name string, cb *circuitbreaker.CircuitBreaker) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.values[name] = cb
}
