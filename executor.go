package cuirass

import (
	"errors"
	"sync"

	"github.com/arjantop/cuirass/circuitbreaker"
)

var UnknownPanic = errors.New("Unknown panic")

// Executor is a main service that knows how to execute commands and handle
// their errors.
// Executor is safe to be accessed by multiple threads.
type Executor struct {
	circuitBreakers cbMap
}

// NewExecutor constructs a new empty executor.
func NewExecutor() *Executor {
	return &Executor{
		circuitBreakers: newCbMap(),
	}
}

// Exec executes a command and handles command execution errors.
// If command fails with an error or panics Fallback function with fallback logic
// is executed. Every command execution is guarded by an internal circuit-breaker.
// Panics are recovered and returned as errors.
func (e *Executor) Exec(cmd Command, result interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = execFallback(cmd, result, r)
		}
	}()
	cb := e.getCircuitBreakerForCommand(cmd)
	// Execute the command in the context of its circuit-breaker.
	err = cb.Do(func() error {
		return cmd.Run(result)
	})
	if err != nil {
		// Panic with error and handle it the same as panic.
		panic(err)
	}
	return nil
}

// executeFallback handles a fallback for a failed command.
// Because a Fallback can panic too errors are recovered the same wasy as for Exec.
func execFallback(cmd Command, result interface{}, r interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicToError(r)
		}
	}()
	err = cmd.Fallback(result)
	if err != nil {
		err = panicToError(r)
	}
	return err
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
	return err
}

// getcircuitbreakerforcommand returns a circuit breaker for a command or constructs
// a new one and returns it.
func (e *Executor) getCircuitBreakerForCommand(cmd Command) *circuitbreaker.CircuitBreaker {
	if cb, ok := e.circuitBreakers.get(cmd.Name()); ok {
		return cb
	} else {
		cb := circuitbreaker.New(
			circuitbreaker.DefaultErrorThreshold,
			circuitbreaker.DefaultSleepWindow,
			circuitbreaker.DefaultRequestVolumeThreshold)
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
